# SQL security: injection-safe dynamic SQL, definer hardening, RLS, privileges

Security is the agent's highest-priority review axis. Maps to OWASP ASI02 (tool
misuse), ASI05 (unexpected code execution — SQL execution is code execution), and
ASI03 (identity/privilege abuse). A draft that is injection-prone or over-
privileged must be flagged as a blocker, not a suggestion.

## SQL injection — the rule

Never build SQL by concatenating untrusted input. Two safe mechanisms:

1. **Bind values as parameters** — the only correct way to pass values:
```sql
EXECUTE 'SELECT * FROM orders WHERE customer_id = $1' USING p_customer;
```
The value is never parsed as SQL; injection is impossible.

2. **Quote identifiers with `format()`** — when a table/column name is dynamic:
```sql
EXECUTE format('SELECT %I FROM %I WHERE id = $1', v_col, v_table) USING p_id;
```
- `%I` — identifier quoting (defeats `"; DROP TABLE ...`).
- `%L` — literal quoting (use only when a bind param truly can't; prefer `$1`).

**Injection-prone (reject in review):**
```sql
EXECUTE 'SELECT * FROM ' || p_table || ' WHERE name = ''' || p_name || '''';
```
Both the identifier and the value are unescaped. This is the classic hole.

Even in static SQL, values from `LIKE` need care: escape `%` and `_` in
user-supplied patterns, or use `ESCAPE`.

## SECURITY DEFINER hardening

A `SECURITY DEFINER` function runs with the owner's privileges — setuid for SQL.
Two mandatory controls:

1. **Pin `search_path`** so a caller can't shadow built-ins/tables:
```sql
CREATE FUNCTION f(...) RETURNS ...
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, public   -- never trust the caller's path
AS $$ ... $$;
```
Without this, a caller creates `public.now()` (or a fake table your function
references unqualified) and executes arbitrary code with elevated rights.

2. **Minimize the owner's privileges** and schema-qualify references inside the
   body. Grant `EXECUTE` on the function only to the roles that need it; `REVOKE`
   from `PUBLIC`.

## Least privilege

- `REVOKE` the default `PUBLIC` grants you don't want (e.g. on new schemas).
- Grant the minimum: `SELECT` where reads suffice, no blanket `ALL`.
- Application roles should not own the objects they use; separate an owner/DDL
  role from the runtime role.
- Use `pg_catalog`-qualified calls in security-sensitive code.

## Row-Level Security (RLS)

For multi-tenant isolation enforced at the data layer:
```sql
ALTER TABLE docs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON docs
  USING (tenant_id = current_setting('app.tenant_id')::bigint);
```
Notes the agent must include when drafting RLS:
- Table owners and `BYPASSRLS` roles bypass policies — verify the runtime role
  does **not** have `BYPASSRLS`.
- `FORCE ROW LEVEL SECURITY` applies policies even to the table owner.
- Separate `USING` (read/visibility) from `WITH CHECK` (write validation).
- Set the tenant key via a trusted mechanism (`SET LOCAL`), never from client-
  controlled input without validation.

## Secrets and data exposure

- Never write credentials, tokens, or PII into a function body, a `RAISE`
  message, or a comment.
- Be deliberate about what a `SECURITY DEFINER` function returns — it can leak
  rows the caller couldn't otherwise see.

## Review checklist (security is a gate, not a nicety)

1. Any dynamic SQL? Is every value a bind param and every identifier `%I`?
2. `SECURITY DEFINER`? Is `search_path` pinned and are grants minimal?
3. Is the runtime role least-privileged and not the object owner?
4. RLS drafts: is `BYPASSRLS` excluded and `WITH CHECK` present for writes?
5. Any secret or PII in code, logs, or comments? (Must be none.)
