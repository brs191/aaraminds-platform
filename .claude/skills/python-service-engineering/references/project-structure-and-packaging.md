# Project Structure and Packaging

This reference covers how to lay out a Python service as a package: the directory layout, module boundaries, the composition root, dependency management, and containerization. It is where a prototype becomes a deployable service.

## A service is a package, not a folder of scripts

The difference between a script and a service is structure you can build on. A service has a defined package, explicit module boundaries, a declared and locked dependency set, and an entry point separate from its logic. A folder of scripts that import each other by relative path, with dependencies installed ad hoc, cannot be tested in isolation, deployed reproducibly, or changed safely. Productionizing a prototype is re-housing its logic into this structure — not packaging the prototype.

## The `src/` layout

Use a `src/` layout: the importable package lives under `src/<package>/`, tests live in a sibling `tests/`. The `src/` layout's value is that it forces the package to be *installed* to be imported, so tests run against the installed package the way production will — it catches "works on my machine because the cwd happens to be right" before it ships. A flat layout (package directory at the repo root) blurs that line.

## Modules — one responsibility each

Split the package into modules by responsibility, not by type-of-thing. A module for the retrieval logic, a module for the orchestration, a module for the model client, a module for configuration — not a `utils.py`, a `models.py`, and a `helpers.py` that everything imports. The test of a good module boundary: it can be described in one sentence and tested without standing up the rest of the service.

## The composition root

Keep one thin entry point — the composition root — that reads configuration, constructs the dependencies (the model client, the retriever, the orchestrator), wires them together, and starts the service. Business logic does not live here; this file only assembles. A composition root keeps the rest of the code free of global state and construction logic, which is what makes modules independently testable. Dependencies are passed in (constructor or function arguments), not reached for as module globals.

## Dependency management — one tool, one lockfile

Pick one dependency manager and commit its lockfile. `uv` is the strong current default — fast, with a proper lockfile and resolver; `poetry` and `pip-tools` are fine alternatives. The non-negotiable is the **lockfile**: the exact resolved versions of every direct and transitive dependency, committed, so every install — a teammate's, CI's, the container build's — is identical. `pip install` against unpinned ranges is how "it worked yesterday" happens. Declare project metadata and dependencies in `pyproject.toml`; it is the standard and every tool reads it.

## Containerization for Container Apps

The service deploys as a container on Azure Container Apps. Use a multi-stage Dockerfile: a build stage that installs dependencies into a virtual environment, and a slim runtime stage (a slim Python base image, or distroless) that copies only the venv and the package — not the build toolchain, not the lockfile-manager. Run as a non-root user. Pin the base image by digest. The result is a small, reproducible image; a single-stage build on a full Python image ships hundreds of megabytes of build tooling as attack surface.

## Notebooks stay out of the production path

A notebook is a good place to explore and a bad place to ship from — it has no module boundaries, hidden cell-execution-order state, and no clean entry point. Prototyping in a notebook is fine; the productionization step is moving the logic into the package structure above, not wrapping or converting the `.ipynb`. A notebook in the deployment path is the structural form of the prototype-to-production anti-pattern in the SKILL.md.

## Verification questions

1. Is the service an installable package with a `src/` layout and a sibling `tests/`?
2. Are modules split by single responsibility — no catch-all `utils`/`helpers`?
3. Is there one thin composition root that wires dependencies, with logic kept out of it and dependencies passed in?
4. Is there one dependency manager with a committed lockfile, and `pyproject.toml` for project metadata?
5. Is the container a multi-stage build on a slim/distroless base, non-root, base image pinned?
6. Is every notebook out of the deployment path?

## What to read next

- `typing-and-pydantic.md` — typing the boundaries inside the structure
- `service-runtime-concerns.md` — what the composition root wires
- `orchestration-code.md` — the orchestration module's shape
- `azure-microservices-observability` — what the runtime image emits
