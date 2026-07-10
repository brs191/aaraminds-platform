# Dynamic SQL and security (sp_executesql, QUOTENAME, RLS, Entra)

Security is the agent's highest-priority review axis. Maps to OWASP ASI02 (tool
misuse), ASI05 (unexpected code execution — dynamic SQL execution is code
execution), ASI03 (identity/privilege abuse). Injection-prone or over-privileged
drafts are blockers, not suggestions.

## Injection-safe dynamic SQL — the rule

Never concatenate untrusted input into a SQL string and `EXEC()` it. The two safe
mechanisms:

1. **Bind values with `sp_executesql`** — typed parameters, never parsed as SQL:
```sql
DECLARE @sql nvarchar(max) = N'SELECT * FROM dbo.Orders WHERE CustomerId = @cust';
EXEC sys.sp_executesql @sql, N'@cust int', @cust = @CustomerId;
```
This also lets the plan cache reuse the query — a performance win over `EXEC()`.

2. **Quote identifiers with `QUOTENAME()`** — when a table/column name is dynamic:
```sql
DECLARE @sql nvarchar(max) =
  N'SELECT * FROM ' + QUOTENAME(@schema) + N'.' + QUOTENAME(@table)
  + N' WHERE ' + QUOTENAME(@col) + N' = @val';
EXEC sys.sp_executesql @sql, N'@val nvarchar(200)', @val = @Value;
```
`QUOTENAME()` wraps in brackets and escapes embedded `]`, defeating identifier
injection.

**Injection-prone (reject in review):**
```sql
EXEC('SELECT * FROM ' + @table + ' WHERE name = ''' + @name + '''');
```
Both the identifier and the value are unescaped — the classic hole. Also flag
`EXEC(@sql)` where `@sql` was built by concatenation.

Even in static SQL, user-supplied `LIKE` patterns need `%`/`_`/`[` escaped via an
`ESCAPE` clause.

## EXECUTE AS and ownership chaining

- **Ownership chaining**: when a procedure and the objects it touches share an
  owner, permission is not re-checked on the inner objects — the caller needs
  `EXECUTE` on the procedure only. This is the normal, safe way to encapsulate
  access; keep objects in one schema owned by one principal.
- **`EXECUTE AS`**: runs the module under a specified principal (`OWNER`,
  `SELF`, a named user). Powerful — the T-SQL analog of setuid. Use `EXECUTE AS
  OWNER` deliberately, grant the owner minimal rights, and never `EXECUTE AS` a
  high-privilege login to paper over a missing grant.

## Row-Level Security (RLS)

Multi-tenant isolation at the engine, via an inline TVF predicate + a security
policy:
```sql
CREATE FUNCTION dbo.fn_tenant_predicate(@TenantId int)
RETURNS TABLE WITH SCHEMABINDING AS
RETURN SELECT 1 AS ok WHERE @TenantId = CAST(SESSION_CONTEXT(N'TenantId') AS int);

CREATE SECURITY POLICY dbo.TenantFilter
  ADD FILTER PREDICATE dbo.fn_tenant_predicate(TenantId) ON dbo.Docs,
  ADD BLOCK PREDICATE dbo.fn_tenant_predicate(TenantId) ON dbo.Docs AFTER INSERT;
```
Notes: use `SESSION_CONTEXT` (set via `sp_set_session_context`) as the trusted
key, never client-supplied input; add a BLOCK predicate for writes, not just a
FILTER for reads; `sysadmin`/`db_owner` and `SCHEMABINDING`-bypass paths need
review.

## Other engine security features

- **Always Encrypted** — encrypts sensitive columns client-side; the engine
  never sees plaintext. Use for regulated data; note it restricts operations
  (equality-only on deterministic encryption).
- **Dynamic Data Masking** — display-layer masking only; not a security boundary
  (a determined user can infer values). Never treat it as encryption.
- **Microsoft Entra authentication** `[AzureSQL]` — prefer Entra (managed
  identities, groups) over SQL logins; disable SQL auth where possible.

## Least privilege

Grant the minimum (`SELECT`/`EXECUTE`, not `db_owner`); the runtime principal
should not own objects; prefer Entra groups; `REVOKE` broad `public` grants.

## Review checklist

1. Any dynamic SQL? Every value via `sp_executesql` params, every identifier via
   `QUOTENAME()`?
2. `EXECUTE AS` used? Is the target principal least-privileged and intentional?
3. RLS drafts: trusted key via `SESSION_CONTEXT`, BLOCK predicate on writes,
   bypass paths reviewed?
4. Is Dynamic Data Masking being mistaken for a security control? (It isn't.)
5. Any secret/PII in code, `THROW`/`RAISERROR` text, or comments? (Must be none.)
