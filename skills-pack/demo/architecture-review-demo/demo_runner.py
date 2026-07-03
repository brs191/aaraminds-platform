#!/usr/bin/env python3
"""Architecture review demo runner.

For each master architecture input (e-commerce, financial-services, healthcare),
this runner spawns the microservices-system-design MCP server as a subprocess
over stdio, calls the five architecture-review tools, and writes their JSON
outputs to per-architecture output directories.

The runner is intentionally stdlib-only: subprocess + json + a minimal hand-rolled
MCP JSON-RPC client. This keeps the demo runnable on any machine with Python 3.8+
without installing the official MCP Python SDK.

All output is produced by the Go MCP server. The runner only shapes per-tool
inputs from the master input and forwards them; it never invents results.
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import threading
from pathlib import Path
from typing import Any

# MCP tools this demo exercises. Each entry maps a short key (used for output
# filenames) to the canonical tool name registered by the Go server.
TOOLS = [
    ("boundary", "generate_service_boundary_canvas"),
    ("apicontract", "generate_api_contract"),
    ("archrisks", "detect_architecture_risks"),
    ("azuremap", "map_patterns_to_azure_services"),
    ("obsplan", "generate_observability_plan"),
]

# MCP protocol version this client speaks. The server is permissive on minor
# versions but pinning here makes mismatches debuggable.
PROTOCOL_VERSION = "2025-06-18"


# ---------------------------------------------------------------------------
# MCP stdio client
# ---------------------------------------------------------------------------


class MCPStdioClient:
    """Minimal MCP client speaking JSON-RPC 2.0 over a subprocess's stdio.

    The MCP spec defines the transport as newline-delimited JSON. This client
    implements just enough of the protocol to:
      1. Spawn the server.
      2. Complete the initialize/notifications/initialized handshake.
      3. Issue tools/call requests and read structured tool results.
      4. Shut down cleanly.

    It is not a full MCP client — no resources, prompts, subscriptions, or
    cancellation. It is enough for the demo runner.
    """

    def __init__(self, server_cmd: list[str]):
        self._server_cmd = server_cmd
        self._proc: subprocess.Popen | None = None
        self._next_id = 1
        self._stderr_thread: threading.Thread | None = None

    # --- Lifecycle -------------------------------------------------------

    def __enter__(self) -> "MCPStdioClient":
        # stderr is forwarded to this process's stderr so server logs are
        # visible during runs (helpful when goldens drift unexpectedly).
        self._proc = subprocess.Popen(
            self._server_cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=0,
            text=False,
        )
        self._stderr_thread = threading.Thread(
            target=self._forward_stderr, daemon=True
        )
        self._stderr_thread.start()
        self._initialize()
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        if self._proc is None:
            return
        try:
            if self._proc.stdin and not self._proc.stdin.closed:
                self._proc.stdin.close()
        except BrokenPipeError:
            pass
        try:
            self._proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            self._proc.terminate()
            self._proc.wait(timeout=2)

    # --- Internal helpers ------------------------------------------------

    def _forward_stderr(self) -> None:
        assert self._proc is not None and self._proc.stderr is not None
        for line in iter(self._proc.stderr.readline, b""):
            try:
                sys.stderr.write("[server] " + line.decode("utf-8", "replace"))
                sys.stderr.flush()
            except Exception:
                break

    def _send(self, payload: dict[str, Any]) -> None:
        assert self._proc is not None and self._proc.stdin is not None
        line = (json.dumps(payload, separators=(",", ":")) + "\n").encode("utf-8")
        self._proc.stdin.write(line)
        self._proc.stdin.flush()

    def _recv(self) -> dict[str, Any]:
        assert self._proc is not None and self._proc.stdout is not None
        raw = self._proc.stdout.readline()
        if not raw:
            raise RuntimeError("MCP server closed stdout before responding")
        return json.loads(raw.decode("utf-8"))

    def _request(self, method: str, params: dict[str, Any]) -> dict[str, Any]:
        msg_id = self._next_id
        self._next_id += 1
        self._send({"jsonrpc": "2.0", "id": msg_id, "method": method, "params": params})
        # MCP servers may emit notifications before the matching response;
        # discard anything that isn't keyed to our id.
        while True:
            msg = self._recv()
            if msg.get("id") == msg_id:
                if "error" in msg:
                    raise RuntimeError(
                        f"MCP error on {method}: {msg['error'].get('message', msg['error'])}"
                    )
                return msg.get("result", {})
            # Ignore unrelated server-initiated notifications.

    def _notify(self, method: str, params: dict[str, Any] | None = None) -> None:
        payload: dict[str, Any] = {"jsonrpc": "2.0", "method": method}
        if params is not None:
            payload["params"] = params
        self._send(payload)

    def _initialize(self) -> None:
        self._request(
            "initialize",
            {
                "protocolVersion": PROTOCOL_VERSION,
                "capabilities": {},
                "clientInfo": {"name": "demo-runner", "version": "0.1.0"},
            },
        )
        self._notify("notifications/initialized")

    # --- Public API ------------------------------------------------------

    def call_tool(self, name: str, arguments: dict[str, Any]) -> str:
        """Call an MCP tool and return its raw text result.

        The architecture-review tools return a single text content block whose
        body is JSON. This method returns that body unparsed so callers can
        decide whether to pretty-print or post-process it.
        """
        result = self._request("tools/call", {"name": name, "arguments": arguments})
        content = result.get("content", [])
        if not content or content[0].get("type") != "text":
            raise RuntimeError(
                f"unexpected tool result shape for {name}: {result!r}"
            )
        if result.get("isError"):
            raise RuntimeError(f"tool {name} returned error: {content[0].get('text')}")
        return content[0].get("text", "")


# ---------------------------------------------------------------------------
# Per-tool input shaping
# ---------------------------------------------------------------------------


def shape_boundary_input(arch: dict[str, Any]) -> dict[str, Any]:
    return {
        "system_name": arch["system_name"],
        "description": arch.get("description", ""),
        "services": [
            {
                "name": s["name"],
                "business_capability": s.get("business_capability", ""),
                "owns_data": s.get("owns_data", []),
                "depends_on": s.get("depends_on", []),
                "consumes_events_from": s.get("consumes_events_from", []),
                "team": s.get("team", ""),
            }
            for s in arch.get("services", [])
        ],
        "data_stores": arch.get("data_stores", []),
        "teams": arch.get("teams", []),
    }


def shape_apicontract_input(arch: dict[str, Any]) -> dict[str, Any]:
    services = []
    for s in arch.get("services", []):
        if not s.get("base_path"):
            # The api-contract tool only cares about services that expose an API.
            continue
        services.append(
            {
                "name": s["name"],
                "business_capability": s.get("business_capability", ""),
                "base_path": s["base_path"],
                "auth": s.get("auth", ""),
                "resources": s.get("resources", []),
            }
        )
    return {
        "system_name": arch["system_name"],
        "description": arch.get("description", ""),
        "api_style": arch.get("api_style", "rest"),
        "versioning_strategy": arch.get("versioning_strategy", "uri"),
        "services": services,
    }


def shape_archrisks_input(arch: dict[str, Any]) -> dict[str, Any]:
    return {
        "system_name": arch["system_name"],
        "description": arch.get("description", ""),
        "deployment_target": arch.get("deployment_target", ""),
        "constraints": arch.get("constraints", []),
        "non_functional_requirements": arch.get("non_functional_requirements", {}),
        "services": [
            {
                "name": s["name"],
                "criticality": s.get("criticality", "medium"),
                "stateful": s.get("stateful", False),
                "replicated": s.get("replicated", True),
                "depends_on": s.get("depends_on", []),
                "consumes_events_from": s.get("consumes_events_from", []),
                "data_stores": s.get("data_stores", []),
                "resilience": s.get("resilience", []),
            }
            for s in arch.get("services", [])
        ],
        "data_stores": arch.get("data_stores", []),
    }


def shape_azuremap_input(arch: dict[str, Any]) -> dict[str, Any]:
    return {
        "system_name": arch["system_name"],
        "description": arch.get("description", ""),
        "deployment_target": arch.get("deployment_target", ""),
        "constraints": arch.get("constraints", []),
        "patterns": arch.get("patterns", []),
    }


def shape_obsplan_input(arch: dict[str, Any]) -> dict[str, Any]:
    return {
        "system_name": arch["system_name"],
        "description": arch.get("description", ""),
        "non_functional_requirements": arch.get("non_functional_requirements", {}),
        "services": [
            {
                "name": s["name"],
                "criticality": s.get("criticality", "medium"),
                "type": s.get("type", "api"),
                "has_dashboards": s.get("has_dashboards", False),
                "has_alerts": s.get("has_alerts", False),
            }
            for s in arch.get("services", [])
        ],
    }


SHAPERS = {
    "boundary": shape_boundary_input,
    "apicontract": shape_apicontract_input,
    "archrisks": shape_archrisks_input,
    "azuremap": shape_azuremap_input,
    "obsplan": shape_obsplan_input,
}


# ---------------------------------------------------------------------------
# Orchestration
# ---------------------------------------------------------------------------


def run_one_architecture(
    client: MCPStdioClient, arch: dict[str, Any], out_dir: Path
) -> None:
    out_dir.mkdir(parents=True, exist_ok=True)
    for short, tool_name in TOOLS:
        tool_input = SHAPERS[short](arch)
        raw = client.call_tool(tool_name, {"input_json": json.dumps(tool_input)})
        # The Go server returns pretty-printed JSON; re-parse and re-emit so we
        # control the indentation in the golden files (canonical comparison).
        parsed = json.loads(raw)
        (out_dir / f"{short}.json").write_text(
            json.dumps(parsed, indent=2, sort_keys=False) + "\n"
        )
        print(f"  - {short} ({tool_name}) -> {out_dir / (short + '.json')}")


def discover_inputs(input_dir: Path) -> list[Path]:
    return sorted(p for p in input_dir.glob("*.json") if p.is_file())


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--server",
        default=os.environ.get("MCP_SERVER_BIN", "mcp-server"),
        help="Path to the MCP server binary (or set MCP_SERVER_BIN).",
    )
    ap.add_argument(
        "--input-dir",
        default="input",
        help="Directory containing master architecture inputs (one JSON per architecture).",
    )
    ap.add_argument(
        "--out",
        default="out",
        help="Output directory; one subdirectory per architecture is written.",
    )
    args = ap.parse_args()

    input_dir = Path(args.input_dir)
    if not input_dir.is_dir():
        print(f"error: input directory not found: {input_dir}", file=sys.stderr)
        return 1
    inputs = discover_inputs(input_dir)
    if not inputs:
        print(f"error: no *.json inputs in {input_dir}", file=sys.stderr)
        return 1

    out_root = Path(args.out)
    out_root.mkdir(parents=True, exist_ok=True)

    with MCPStdioClient([args.server]) as client:
        for input_path in inputs:
            arch_name = input_path.stem
            arch = json.loads(input_path.read_text())
            print(f"\n[{arch_name}] {arch.get('system_name', '?')}")
            run_one_architecture(client, arch, out_root / arch_name)

    print(f"\nGenerated outputs in {out_root} for {len(inputs)} architecture(s).")
    return 0


if __name__ == "__main__":
    sys.exit(main())
