# CI/CD Pipeline — GitHub Actions with OIDC for Azure

The standard CI/CD pipeline for a new service. GitHub Actions, OIDC federation to Entra ID (no long-lived service principal secrets), Terraform for infra, Container Apps revisions for blue-green deploy.

## The three workflows

Every service has exactly three workflows under `.github/workflows/`:

1. **`pr.yml`** — runs on pull request: lint, test, build image to PR-scoped tag, infra plan
2. **`main.yml`** — runs on push to `main`: build image to release-candidate tag, deploy to staging via OIDC, run smoke tests, gate production deploy on manual approval
3. **`release.yml`** — runs on git tag (`v*`): deploy to production via OIDC, post-deploy validation

This is **not** GitOps. We're not running ArgoCD or Flux. Deploys are explicit pipeline runs. This is appropriate for a personal / small-team setup; reach for GitOps when the estate grows past ~30 services and per-service pipelines become unmanageable.

## OIDC setup (one-time)

GitHub OIDC federates to Entra ID without storing client secrets. Setup per environment:

```bash
# 1. Create an Entra ID app registration (Azure CLI):
az ad app create --display-name "github-deployer-<svc>-staging"

# 2. Create a service principal:
az ad sp create --id <app-id>

# 3. Add federated credentials for the GitHub repo + environment:
az ad app federated-credential create --id <app-id> --parameters '{
  "name": "github-staging",
  "issuer": "https://token.actions.githubusercontent.com",
  "subject": "repo:<org>/<svc>:environment:staging",
  "audiences": ["api://AzureADTokenExchange"]
}'

# 4. Grant the SP RBAC on the target subscription / resource group:
az role assignment create \
  --assignee <app-id> \
  --role "Container Apps Contributor" \
  --scope /subscriptions/<sub>/resourceGroups/rg-<svc>-staging
```

In the workflow, no `AZURE_CREDENTIALS` secret is set. The OIDC dance happens automatically via `azure/login@v2`. This eliminates the "rotate service principal secrets quarterly" hazard.

## `pr.yml` — PR gate

```yaml
name: PR

on:
  pull_request:
    branches: [main]

permissions:
  contents: read
  id-token: write          # Required for OIDC

jobs:
  lint-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # Language-specific (Go example; swap for Java setup-java + mvn)
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }

      - name: Lint
        run: |
          gofmt -l . | tee /dev/stderr | (! grep .)
          go vet ./...
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

      - name: Test
        run: go test -race -count=1 ./...

      - name: Build
        run: go build ./...

  docker-build:
    needs: lint-test
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      contents: read
    steps:
      - uses: actions/checkout@v4
      - uses: azure/login@v2
        with:
          client-id: ${{ vars.AZURE_CLIENT_ID_STAGING }}
          tenant-id: ${{ vars.AZURE_TENANT_ID }}
          subscription-id: ${{ vars.AZURE_SUBSCRIPTION_ID }}
      - uses: azure/docker-login@v1
        with:
          login-server: ${{ vars.ACR_LOGIN_SERVER }}
      - name: Build image
        run: |
          IMG=${{ vars.ACR_LOGIN_SERVER }}/<svc>:pr-${{ github.event.pull_request.number }}
          docker build -t $IMG .
          docker push $IMG

  infra-plan:
    needs: lint-test
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      contents: read
    steps:
      - uses: actions/checkout@v4
      - uses: azure/login@v2
        with:
          client-id: ${{ vars.AZURE_CLIENT_ID_STAGING }}
          tenant-id: ${{ vars.AZURE_TENANT_ID }}
          subscription-id: ${{ vars.AZURE_SUBSCRIPTION_ID }}
      - uses: hashicorp/setup-terraform@v3
      - name: Terraform plan
        working-directory: infra
        run: |
          terraform init
          terraform plan -out=tfplan
```

PR gate budget: **under 5 minutes** total. If it slips above that, see `mcp-go-production-review`'s gate hierarchy advice — slow PR gates get routed around.

## `main.yml` — staging deploy on merge

```yaml
name: main

on:
  push:
    branches: [main]

permissions:
  contents: read
  id-token: write

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    outputs:
      image-tag: ${{ steps.tag.outputs.tag }}
    steps:
      - uses: actions/checkout@v4
      - uses: azure/login@v2
        with:
          client-id: ${{ vars.AZURE_CLIENT_ID_STAGING }}
          tenant-id: ${{ vars.AZURE_TENANT_ID }}
          subscription-id: ${{ vars.AZURE_SUBSCRIPTION_ID }}
      - uses: azure/docker-login@v1
        with:
          login-server: ${{ vars.ACR_LOGIN_SERVER }}
      - id: tag
        run: echo "tag=$(date +%Y%m%d)-${GITHUB_SHA::7}" >> $GITHUB_OUTPUT
      - run: |
          IMG=${{ vars.ACR_LOGIN_SERVER }}/<svc>:${{ steps.tag.outputs.tag }}
          docker build -t $IMG .
          docker push $IMG

  deploy-staging:
    needs: build-and-push
    runs-on: ubuntu-latest
    environment: staging        # Requires environment-level OIDC config
    steps:
      - uses: actions/checkout@v4
      - uses: azure/login@v2
        with:
          client-id: ${{ vars.AZURE_CLIENT_ID_STAGING }}
          tenant-id: ${{ vars.AZURE_TENANT_ID }}
          subscription-id: ${{ vars.AZURE_SUBSCRIPTION_ID }}
      - name: Update Container Apps revision
        run: |
          az containerapp update \
            --name <svc>-staging \
            --resource-group rg-<svc>-staging \
            --image ${{ vars.ACR_LOGIN_SERVER }}/<svc>:${{ needs.build-and-push.outputs.image-tag }}
      - name: Smoke test
        run: |
          curl -fsS https://<svc>-staging.azurewebsites.net/healthz
          # Run additional smoke tests here

  integration-tests:
    needs: deploy-staging
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - name: Integration tests against staging
        env:
          STAGING_URL: https://<svc>-staging.azurewebsites.net
        run: go test -tags=integration ./...
```

## `release.yml` — production deploy on git tag

```yaml
name: release

on:
  push:
    tags: ['v*']

permissions:
  contents: read
  id-token: write

jobs:
  deploy-prod:
    runs-on: ubuntu-latest
    environment: production       # Manual approval gate via GitHub Environments
    steps:
      - uses: actions/checkout@v4
      - uses: azure/login@v2
        with:
          client-id: ${{ vars.AZURE_CLIENT_ID_PROD }}
          tenant-id: ${{ vars.AZURE_TENANT_ID }}
          subscription-id: ${{ vars.AZURE_SUBSCRIPTION_ID }}

      - name: Deploy new revision (canary 10%)
        run: |
          IMG=${{ vars.ACR_LOGIN_SERVER }}/<svc>:${GITHUB_REF_NAME#v}
          az containerapp revision copy \
            --name <svc>-prod \
            --resource-group rg-<svc>-prod \
            --image $IMG \
            --revision-suffix ${{ github.run_id }}
          az containerapp ingress traffic set \
            --name <svc>-prod \
            --resource-group rg-<svc>-prod \
            --label-weight new=10 stable=90

      - name: Wait for canary signals
        run: sleep 600                    # 10 minutes baseline; tune for the service

      - name: Promote canary to 100%
        # In a real pipeline, gate this on SLO metrics (e.g., a manual approval or an auto-check
        # against Prometheus / Application Insights)
        run: |
          az containerapp ingress traffic set \
            --name <svc>-prod \
            --resource-group rg-<svc>-prod \
            --label-weight new=100 stable=0
```

The canary-then-promote pattern uses Container Apps revisions. See `../../microservices-resilience/references/patterns/blue-green-canary.md` for the underlying rollout pattern.

## Environment-scoped OIDC

The pattern above uses GitHub Environments (`environment: staging` / `environment: production`) which:

- Require approval before the job runs (configurable per environment — typically required for production, not for staging)
- Scope OIDC token issuance to the environment subject claim — the staging app registration only accepts tokens scoped to the staging environment
- Provide environment-scoped secrets if absolutely needed (avoid; use Key Vault)

This means a successful PR-gate workflow cannot deploy to production. Only the `release.yml` workflow with the `production` environment can.

## Terraform usage

Terraform runs from the pipeline using the same OIDC identity. The pipeline does NOT have terraform state credentials stored — state lives in Azure Storage with RBAC granted to the deploy SP.

```hcl
# infra/backend.tf
terraform {
  backend "azurerm" {
    resource_group_name  = "rg-tfstate"
    storage_account_name = "stataaramindsstate"
    container_name       = "<svc>"
    key                  = "terraform.tfstate"
    use_oidc             = true       # Federated identity to the storage account
  }
}
```

`use_oidc = true` makes Terraform pick up the OIDC token issued to the workflow. No Storage account access keys, no state lock account, no service principal secret for state.

## What `pr.yml` runs in under 5 minutes (budget breakdown)

- Checkout: 5 s
- Go install: 15 s (cached)
- gofmt: 2 s
- go vet: 5 s
- govulncheck: 15 s
- go test -race: 30-90 s (workload-dependent)
- Docker build: 30-60 s (multi-stage, cached layers)
- Docker push to ACR: 5-30 s (image size dependent)
- Terraform plan: 30 s

Total: 2-4 minutes for a typical service. If you find yourself above 5 min, the cure is to move slow checks (integration tests against external systems, image scans against fresh CVE DBs, demo runs) to `main.yml` post-merge.

## Anti-patterns (violations of the standard CI shape)

- **Long-lived `AZURE_CREDENTIALS` secret** — must use OIDC; no service principal client secret in `secrets`.
- **`master`/`main` deploys auto-promoting to production** — production deploy is on tag with manual approval. Auto-promote to staging is fine; auto-promote to production is the recipe for the "git push, customer outage" incident.
- **State stored in the repo or in pipeline secrets** — Terraform state in Azure Storage, locked, with state-account RBAC.
- **Docker Hub instead of ACR** — use Azure Container Registry; ACR has Entra ID auth, vulnerability scanning, geo-replication, and is the only registry that integrates cleanly with Container Apps and Defender for Containers.
- **`make deploy` from a developer laptop to production** — production deploys go through the pipeline. Period. The pipeline records what was deployed, when, by whom, with which infrastructure plan, in a way `make deploy` from a laptop never can.

## Verification

A new service's CI/CD passes the standard check if:

1. Three workflows exist (`pr.yml`, `main.yml`, `release.yml`)
2. All three use `permissions: id-token: write` and `azure/login@v2` with OIDC
3. Zero `AZURE_CREDENTIALS` / `AZURE_CLIENT_SECRET` / `ARM_CLIENT_SECRET` references in any workflow or secret
4. Production environment requires manual approval via GitHub Environments
5. PR-gate budget under 5 minutes
6. Terraform state is in Azure Storage with `use_oidc = true`
7. Container image hosted in Azure Container Registry, not Docker Hub
8. Production deploy uses Container Apps revisions with canary-then-promote pattern
