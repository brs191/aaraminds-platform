# Layer 1 — Ingress Authentication

> Establish *who* the caller is at the edge, then re-establish it inside the service. Authenticate every ingress with Microsoft Entra ID and OAuth 2.1; never trust a header an upstream proxy claims to have validated.

## The rule

Validate the token twice: a cheap reject at the edge (Azure API Management `validate-jwt`) and an authoritative check in the service (resource-server validation). The edge stops the obvious garbage and protects the backend; the in-service check is the one you actually trust, because it survives a misconfigured or bypassed gateway. Defense in depth means the service does not assume the gateway did its job.

## OAuth 2.1 flows (what changed from 2.0)

| Caller | Flow | Notes |
|---|---|---|
| SPA, mobile, server-rendered web | Authorization Code **+ PKCE** | PKCE is mandatory in 2.1; the implicit flow and password grant are removed — do not use them |
| Daemon / service with no user | Client Credentials | The caller is a workload, not a person; prefer Managed Identity over a client secret (see `authorization-and-service-identity.md`) |
| Partner B2B system | Client Credentials or mTLS | Use mTLS when the partner cannot hold an Entra credential |

There is no legitimate new-build reason to use the implicit flow. If you find it, it is brownfield to migrate.

## Entra ID specifics

- **App registration** per API. Expose scopes under an Application ID URI (`api://<app-id>/Orders.Read`) and define **app roles** for application (daemon) permissions.
- Use the **v2.0 endpoint**; pin `accessTokenAcceptedVersion: 2`. v1.0 tokens carry different claim shapes and audiences.
- Delegated (user) permissions surface in the `scp` claim; application permissions surface in `roles`. Authorization logic must read the right one — see `authorization-and-service-identity.md`.

## Token validation — the five checks

Every token, at every service:

1. **Signature** against the tenant JWKS (`https://login.microsoftonline.com/<tenant>/discovery/v2.0/keys`); cache keys, honor `kid` rollover.
2. **Issuer** (`iss`) matches your tenant's v2.0 issuer exactly.
3. **Audience** (`aud`) equals *your* API's app ID / URI. This is the check teams skip — validating signature but not audience accepts a valid token minted for a *different* API.
4. **Expiry / not-before** (`exp`, `nbf`) with small clock-skew tolerance (≤120s).
5. **Scope / role** (`scp` or `roles`) authorizes the specific operation.

### Java (Spring Boot resource server)

```java
// build.gradle: implementation 'org.springframework.boot:spring-boot-starter-oauth2-resource-server'
// application.yml
spring:
  security:
    oauth2:
      resourceserver:
        jwt:
          issuer-uri: https://login.microsoftonline.com/<tenant-id>/v2.0
          audiences: api://<app-id>
```

```java
@PreAuthorize("hasAuthority('SCOPE_Orders.Read')")
public Order get(String id) { ... }
```

Spring validates signature, issuer, audience, and expiry from `issuer-uri` + `audiences`; you enforce scope per method.

### Go

```go
provider, _ := oidc.NewProvider(ctx, "https://login.microsoftonline.com/<tenant-id>/v2.0")
verifier := provider.Verifier(&oidc.Config{ClientID: "api://<app-id>"}) // checks aud
idToken, err := verifier.Verify(ctx, rawToken)                          // checks sig, iss, exp
```

`github.com/coreos/go-oidc` + `golang.org/x/oauth2`. Verify audience explicitly; do not rely on signature alone.

## API Management as the front door

Put APIM in front for the cheap edge reject and uniform policy:

```xml
<validate-jwt header-name="Authorization" failed-validation-httpcode="401">
  <openid-config url="https://login.microsoftonline.com/<tenant-id>/v2.0/.well-known/openid-configuration" />
  <audiences><audience>api://<app-id></audience></audiences>
  <issuers><issuer>https://login.microsoftonline.com/<tenant-id>/v2.0</issuer></issuers>
</validate-jwt>
```

APIM **subscription keys are not authentication** — they identify a product subscription for rate-limiting and quotas, nothing more. Never let a subscription key stand in for a validated identity.

## mTLS for partner ingress

Terminate client-certificate auth at Application Gateway or APIM; validate the chain to a known CA and check the certificate thumbprint/subject. Rotate partner certs on a schedule and alert before expiry. Use mTLS only at the B2B boundary — inside the estate, service identity (Managed Identity / mesh) is the right tool, not per-service certs you have to manage.

## When API keys are acceptable (and their limits)

Acceptable for low-trust internal callers or machine clients where standing up OAuth is genuinely disproportionate. But a key carries **no identity, no expiry, no scope** by default. If you use one: store it in Key Vault, inject via Managed Identity, rate-limit per key, rotate on a schedule, and never make it the *only* control. A key is a stopgap, not a security design.

## Failure modes

- **Audience not checked** → a token minted for another API in the same tenant is accepted. The single most common ingress-auth defect.
- **Wrong tenant accepted** → multi-tenant misconfiguration lets external tenants in. Pin the issuer.
- **Trusting `X-Forwarded-*` / `X-User` from the gateway** without re-validating the token in the service → anyone who reaches the pod past the gateway is "authenticated."
- **No JWKS key-rollover handling** → outage when Entra rotates signing keys; cache with refresh, don't hardcode.

## Brownfield: fronting a legacy service

Stand up APIM in front, enable `validate-jwt` at the edge, and run the legacy service in parallel behind it. Then push validation *into* the service (resource-server library) and only once the in-service check is green, remove any implicit trust of gateway headers. Cut over per-route, not big-bang.

## Read next

- `authorization-and-service-identity.md` — what the authenticated caller may do, and how services authenticate each other
- `references/patterns/zero-trust-service-access.md` — the end-to-end zero-trust recipe
