# Terraform AzureRM Anti-Patterns to Flag in PR Review

Named patterns to scan for when reviewing a Terraform PR (AzureRM provider). All targeted at the pack's standardized stack: AzureRM with RBAC, Container Apps, Managed Identity, Key Vault, GitHub Actions OIDC for state.

## 1. Plaintext secrets in `.tf` files

**Pattern:**
```hcl
resource "azurerm_postgresql_flexible_server" "main" {
  administrator_password = "Hunter2!"
}
```

**Detection cue:** any string value in a `.tf` file that looks like a password, GUID, base64 blob, or "secret-shaped."

**Why it fails:** Terraform state stores the value plaintext; the repo stores it plaintext. Both are leak surfaces.

**Fix:** Key Vault data source + reference:
```hcl
data "azurerm_key_vault_secret" "pg_admin_password" {
  name         = "pg-admin-password"
  key_vault_id = data.azurerm_key_vault.main.id
}

resource "azurerm_postgresql_flexible_server" "main" {
  administrator_password = data.azurerm_key_vault_secret.pg_admin_password.value
}
```

The value still ends up in state — that's why the state account itself is locked to deploy-identity-only access (see anti-pattern #6).

## 2. Service principal with long-lived secret

**Pattern:**
```hcl
resource "azurerm_application" "deployer" {
  display_name = "github-deployer"
}

resource "azurerm_application_password" "deployer_secret" {
  application_id = azurerm_application.deployer.id
  end_date       = "2099-12-31T23:59:59Z"
}
```

**Detection cue:** `azurerm_application_password` resource, or `azuread_service_principal_password`.

**Why it fails:** long-lived secrets rotate poorly, leak through CI logs, and require manual rotation discipline. The whole reason GitHub Actions OIDC exists is to eliminate them.

**Fix:** federated credentials:
```hcl
resource "azuread_application_federated_identity_credential" "github" {
  application_id = azuread_application.deployer.id
  display_name   = "github-actions-staging"
  audiences      = ["api://AzureADTokenExchange"]
  issuer         = "https://token.actions.githubusercontent.com"
  subject        = "repo:${var.github_org}/${var.repo_name}:environment:staging"
}
```

## 3. Over-scoped role assignments

**Pattern:**
```hcl
resource "azurerm_role_assignment" "deployer_owner" {
  scope                = "/subscriptions/${var.subscription_id}"
  role_definition_name = "Owner"
  principal_id         = azuread_service_principal.deployer.object_id
}
```

**Detection cue:** `role_definition_name = "Owner"` or `"Contributor"` at subscription scope.

**Why it fails:** the deployer gets way more authority than it needs. Compromise of the deployer = compromise of the subscription.

**Fix:** scope to the resource group:
```hcl
resource "azurerm_role_assignment" "deployer_ca_contrib" {
  scope                = azurerm_resource_group.svc.id
  role_definition_name = "Container Apps Contributor"   # Service-specific role
  principal_id         = azuread_service_principal.deployer.object_id
}
```

Use the smallest built-in role that lets the deploy succeed. Build custom roles if the built-ins are too broad.

## 4. `lifecycle { ignore_changes = all }`

**Pattern:**
```hcl
resource "azurerm_container_app" "main" {
  # ...
  lifecycle {
    ignore_changes = all
  }
}
```

**Detection cue:** `ignore_changes = all` or `ignore_changes = ["tags", "everything"]`.

**Why it fails:** the resource has drifted from IaC; future changes are unpredictable. "We don't know what's in production" is the operational consequence.

**Fix:** identify the specific attributes that drift and `ignore_changes` only those, with a comment explaining why:
```hcl
lifecycle {
  ignore_changes = [
    template[0].container[0].image,    # Image tag is updated by the deploy pipeline
  ]
}
```

## 5. Module sources not pinned

**Pattern:**
```hcl
module "container_app" {
  source = "github.com/aaraminds/terraform-azurerm-container-app"
}
```

**Detection cue:** `source = "github.com/..."` or `source = "git::..."` without `?ref=...`.

**Why it fails:** pulls the default branch; a downstream commit changes your infrastructure without your code changing. Reproducibility broken.

**Fix:** pin to a tag or SHA:
```hcl
module "container_app" {
  source = "github.com/aaraminds/terraform-azurerm-container-app?ref=v1.4.0"
}
```

## 6. `local` state backend

**Pattern:**
```hcl
terraform {
  # No backend block, or:
  backend "local" {}
}
```

**Detection cue:** missing `backend "azurerm"` block, or explicit `local` backend.

**Why it fails:** state lives on the developer's laptop; no team coordination; no audit trail; merging changes requires copying state files around.

**Fix:**
```hcl
terraform {
  backend "azurerm" {
    resource_group_name  = "rg-tfstate"
    storage_account_name = "stataaramindsstate"
    container_name       = "<svc>"
    key                  = "terraform.tfstate"
    use_oidc             = true
  }
}
```

`use_oidc = true` reads the OIDC token from the GitHub Actions runner; no storage account key needed.

## 7. Resources without tags

**Pattern:** any `azurerm_*` resource block without a `tags = { ... }` block.

**Detection cue:** look for `azurerm_resource_group`, `azurerm_container_app`, `azurerm_postgresql_flexible_server`, etc., without a `tags` attribute.

**Why it fails:** cost attribution (`azure-microservices-cost-review` requires resource-group tags); ownership lookup ("who owns this resource") becomes a manual investigation; lifecycle management (`environment = "dev"` tags drive auto-shutdown / cleanup automation).

**Fix:** consistent tags on every resource:
```hcl
locals {
  common_tags = {
    environment = var.environment      # dev | staging | prod
    owner       = var.owning_team      # platform-team
    cost_center = var.cost_center      # CC1234
    service     = "<svc>"
    iac         = "terraform"
  }
}

resource "azurerm_resource_group" "svc" {
  name     = "rg-${var.environment}-<svc>"
  location = var.location
  tags     = local.common_tags
}
```

## 8. Hardcoded locations

**Pattern:** `location = "eastus"` in resource blocks.

**Detection cue:** any literal Azure region in a `location` attribute of a non-variable resource.

**Why it fails:** can't reuse the module across regions; can't do disaster-recovery deploys without forking; review of "is this in the right region" becomes manual.

**Fix:** `location = var.location` with a variable that has reasonable defaults per environment.

## 9. `count` for resource singletons

**Pattern:**
```hcl
resource "azurerm_postgresql_flexible_server" "main" {
  count = var.environment == "prod" ? 1 : 0
  ...
}
```

**Detection cue:** `count = X ? 1 : 0` patterns.

**Why it fails:** `for_each` is cleaner and gives stable resource keys. `count` changes mean resources get re-created when the array shifts.

**Fix:** `for_each` with a map:
```hcl
resource "azurerm_postgresql_flexible_server" "main" {
  for_each = var.deploy_db ? { primary = {} } : {}
  ...
}
```

Or just guard at module-input level: don't include the module in environments that don't need it.

## 10. `null_resource` with `local-exec`

**Pattern:**
```hcl
resource "null_resource" "run_migrations" {
  provisioner "local-exec" {
    command = "psql ... -f migrations.sql"
  }
}
```

**Detection cue:** `null_resource` with `local-exec` or `remote-exec` provisioners.

**Why it fails:** provisioners run only on resource create/destroy, not on update; they break Terraform's idempotency model; they depend on local tooling (psql) being installed.

**Fix:** run migrations from the application or a separate pipeline step, not Terraform. Terraform provisions infrastructure; database schema migration is application-layer concern.

## 11. Diagnostic settings not configured

**Pattern:** Container Apps / Postgres / Key Vault deployed without `azurerm_monitor_diagnostic_setting`.

**Detection cue:** new resource block for a service that produces audit/operational logs, with no matching diagnostic settings sending logs to Log Analytics or Storage.

**Why it fails:** when an incident happens, the logs needed for forensics aren't available. SOC 2 / ISO 27001 audit-log requirements aren't met. See `soc2-iso27001-controls-mapping` skill.

**Fix:**
```hcl
resource "azurerm_monitor_diagnostic_setting" "ca" {
  name                       = "<svc>-diag"
  target_resource_id         = azurerm_container_app.main.id
  log_analytics_workspace_id = data.azurerm_log_analytics_workspace.main.id

  enabled_log { category_group = "audit" }
  enabled_log { category_group = "allLogs" }
  metric { category = "AllMetrics" }
}
```

## 12. Public network access on data resources

**Pattern:**
```hcl
resource "azurerm_postgresql_flexible_server" "main" {
  public_network_access_enabled = true       # Or just not set; default is true on some resources
}
```

**Detection cue:** `public_network_access_enabled = true`, or absence of the attribute on data-tier resources where the default is permissive.

**Why it fails:** data tier reachable from the public internet. Combined with weak auth, this is a primary breach path. Defense-in-depth says: defense at the network layer is a free additional layer.

**Fix:** disable public access, use private endpoints:
```hcl
resource "azurerm_postgresql_flexible_server" "main" {
  public_network_access_enabled = false
  delegated_subnet_id           = azurerm_subnet.pg.id
  private_dns_zone_id           = azurerm_private_dns_zone.pg.id
}
```

## 13. Storage account without encryption / soft-delete

**Pattern:** `azurerm_storage_account` without explicit encryption / soft-delete config.

**Detection cue:** check that the block has `infrastructure_encryption_enabled = true`, `blob_properties { delete_retention_policy { days = 30 } }`, and (where applicable) `customer_managed_key`.

**Why it fails:** encryption defaults are usually OK, but compliance audits want explicit configuration. Soft-delete defaults are *not* enabled; a delete is permanent without it.

**Fix:**
```hcl
resource "azurerm_storage_account" "main" {
  ...
  infrastructure_encryption_enabled = true
  blob_properties {
    delete_retention_policy { days = 30 }
    container_delete_retention_policy { days = 30 }
  }
}
```

## 14. Mixing environments in one state

**Pattern:** a single `.tf` file that deploys dev + staging + prod via `count = var.env == "..." ? 1 : 0`.

**Detection cue:** `var.environment` used to gate resource counts in a single state.

**Why it fails:** changing dev infrastructure shouldn't require running `terraform apply` against a state that includes production. A bug or wrong variable can damage prod.

**Fix:** separate state per environment. One Terraform invocation per environment; environment-specific `.tfvars` files; environment-specific state keys in the azurerm backend.

## 15. Diff-incompatible variable changes

**Pattern:** modifying a variable's `type` or removing it from `variables.tf` without a state migration plan.

**Detection cue:** review the variable diff — anything other than adding a new variable, or relaxing a constraint, may break consumers.

**Why it fails:** downstream `.tfvars` files or module callers break.

**Fix:** treat variable schemas as a contract. Add new variables with defaults; deprecate old ones (mark in comments) before removing in a future major version.

## How to use this list in review

For any Terraform PR: open the diff, search for these patterns. The fastest signals to grep: `password = "`, `azurerm_application_password`, `role_definition_name = "Owner"`, `ignore_changes = all`, `?ref=` missing in module sources, `backend "local"`, missing `tags`, hardcoded region strings, `public_network_access_enabled = true`.

Run `tflint` and `tfsec` in CI to catch the static-checkable subset automatically. Treat this list as the human-judgment layer.
