# Checkov — the adopted security baseline

Stage 1. Checkov (Apache-2.0, Prisma/Bridgecrew) ships thousands of maintained checks and does graph-based cross-resource analysis. Adopt it; do not re-implement security checks in antr.

## Run it over BOTH HCL and plan JSON

- Directory/HCL scan (fast, pre-plan): `checkov -d . --compact`.
- **Plan-JSON scan (authoritative):** `terraform plan -out tfplan.binary && terraform show -json tfplan.binary > tfplan.json && checkov -f tfplan.json`. Plan JSON resolves variables, module inputs, and computed values that a raw `.tf` scan cannot see — this is where real misconfigs surface.
- Azure checks are namespaced `CKV_AZURE_*` (e.g. storage public access, TLS minimums, NSG open management ports, Key Vault soft-delete/purge-protection). Checkov's graph checks can catch patterns like "resource in a public subnet with a permissive rule" — but note these are **config patterns, not true reachability** (that is the analyzer's job).

## Triage and suppress responsibly

- Gate on severity: fail CI on HIGH/CRITICAL by default; report MEDIUM/LOW.
- Suppress with intent, never blanket: inline `# checkov:skip=CKV_AZURE_NN: <reason, owner, ticket>` on the specific resource. A bare `--skip-check` list with no justification erodes the baseline and hides regressions.
- Output SARIF (`-o sarif`) for code-scanning/PR annotations; output JUnit for the CI gate.

## Custom Checkov policies (when Rego is overkill)

For simple attribute assertions you can write Checkov custom policies (Python or YAML graph) instead of Rego — keep org rules in one place. Prefer OPA/Conftest (stage 2) when the logic is relational or needs data documents.

## Pin for reproducibility

Pin the Checkov version and any external policy bundle; a floating version makes the gate non-deterministic and silently changes pass/fail between runs — the same determinism discipline the engine follows.

## Done when

Checkov runs over plan JSON in CI, fails on HIGH/CRITICAL, emits SARIF + JUnit, suppressions are per-resource and justified, and the version is pinned.
