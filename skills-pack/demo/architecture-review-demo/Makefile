# Architecture review demo
#
# Targets:
#   make demo       Run the demo across all input architectures
#   make validate   Compare generated outputs to golden fixtures
#   make refresh    Promote current outputs to goldens (use only when changes are intended)
#   make clean      Remove generated outputs
#
# Requires MCP_SERVER_BIN to point at a built mcp-server binary.
# Build it once from ../../examples/microservices-system-design-mcp-server:
#
#   cd ../../examples/microservices-system-design-mcp-server
#   go build -buildvcs=false -o /tmp/mcp-bin/mcp-server ./cmd/server
#
# Then set MCP_SERVER_BIN=/tmp/mcp-bin/mcp-server (or pass it on each invocation).

MCP_SERVER_BIN ?= ../../examples/microservices-system-design-mcp-server/mcp-server
INPUT_DIR ?= input
OUT_DIR ?= out
GOLDEN_DIR ?= golden

.PHONY: demo validate refresh clean

demo:
	@test -x "$(MCP_SERVER_BIN)" || (echo "error: MCP_SERVER_BIN not executable: $(MCP_SERVER_BIN)"; exit 1)
	python3 demo_runner.py --server "$(MCP_SERVER_BIN)" --input-dir "$(INPUT_DIR)" --out "$(OUT_DIR)"

validate:
	python3 validate_outputs.py --out "$(OUT_DIR)" --golden "$(GOLDEN_DIR)"

refresh:
	@test -d "$(OUT_DIR)" || (echo "error: $(OUT_DIR) does not exist; run 'make demo' first"; exit 1)
	rm -rf "$(GOLDEN_DIR)"
	cp -r "$(OUT_DIR)" "$(GOLDEN_DIR)"
	@echo "Goldens refreshed from $(OUT_DIR)"

clean:
	rm -rf "$(OUT_DIR)"
