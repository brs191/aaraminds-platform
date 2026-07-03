# CI gating on the Terraform plan JSON

Stage 3. The policy gate is only real if it blocks the PR. Wire it as a required check in the same pipeline that runs the engine's reachability and diagram-eval gates.

## Canonical pipeline order

```
terraform init
terraform validate
terraform plan -out tfplan.binary           # resolves vars/modules/computed
terraform show -json tfplan.binary > tfplan.json
checkov   -f tfplan.json -o sarif -o junitxml   # baseline (HIGH/CRITICAL block)
conftest  test tfplan.json                       # custom org policy (deny blocks)
# ── only on green: the reachability gate (separate skill) ──
```

- Both Checkov and Conftest must be **required** status checks; a soft/advisory gate is decoration.
- Emit SARIF for PR code-scanning annotations and JUnit for the pass/fail signal and history.
- Run the policy gate **before** the reachability/analyzer gate is cheap, but the two are independent — a failure in either blocks the merge.

## Reproducibility and OIDC

- Pin tool versions and policy-bundle refs (container digests or exact versions) — the gate's verdict must be reproducible for the same plan.
- Auth to Azure for `terraform plan` via **OIDC federated identity**, read scope only — never `AZURE_CLIENT_SECRET` (AaraMinds standard). Generation emits a PR; CI does not `terraform apply`.

## Performance

Cache the provider plugins and the policy bundle. For large estates, scope the plan to the changed module so the gate stays fast enough to be a required check.

## Done when

Checkov + Conftest run over plan JSON as **required** checks, emit SARIF + JUnit, are version-pinned, authenticate via OIDC read-only, and block the PR on any HIGH/CRITICAL or deny.
