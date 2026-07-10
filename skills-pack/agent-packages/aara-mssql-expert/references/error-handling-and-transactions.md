# Error handling and transactions (TRY/CATCH, THROW, XACT_ABORT)

SQL Server error handling is `TRY...CATCH` — not exception blocks. Get the
transaction/rollback discipline right or a failed procedure leaves an open,
doomed transaction.

## The canonical pattern

```sql
CREATE OR ALTER PROCEDURE dbo.TransferFunds
  @from int, @to int, @amount decimal(19,4)
AS
BEGIN
  SET NOCOUNT ON;
  SET XACT_ABORT ON;               -- doomed transactions roll back automatically
  BEGIN TRY
    BEGIN TRAN;
      UPDATE dbo.Accounts SET balance = balance - @amount WHERE id = @from;
      UPDATE dbo.Accounts SET balance = balance + @amount WHERE id = @to;
    COMMIT;
  END TRY
  BEGIN CATCH
    IF XACT_STATE() <> 0 ROLLBACK;  -- roll back if a transaction is active/doomed
    THROW;                          -- re-raise the original error to the caller
  END CATCH
END;
```

Key points:
- `SET XACT_ABORT ON` ensures most runtime errors abort the batch and mark the
  transaction uncommittable, so `ROLLBACK` in `CATCH` cleans up reliably.
- `XACT_STATE()` returns 1 (active, committable), -1 (active, doomed —
  rollback only), or 0 (none). Check it before `COMMIT`/`ROLLBACK`.
- `THROW` with no arguments re-raises the caught error preserving number and
  severity. Prefer `THROW` over `RAISERROR` for new code.

## THROW vs RAISERROR

- `THROW` (preferred): `THROW 50001, 'Order not found', 1;`. Simpler, always
  severity 16, re-raise with bare `THROW;`. Statement before `THROW` needs a
  semicolon.
- `RAISERROR` (legacy): supports `printf`-style formatting and custom severity,
  but does not by itself abort the batch. Use only when you need formatted
  messages or a specific severity/state.

## ERROR_* functions (inside CATCH)

`ERROR_NUMBER()`, `ERROR_MESSAGE()`, `ERROR_SEVERITY()`, `ERROR_STATE()`,
`ERROR_LINE()`, `ERROR_PROCEDURE()` — capture these for logging before `THROW`.

## What TRY/CATCH does NOT catch

- Compile errors and object-name resolution errors in the same batch.
- Severity 20+ connection-terminating errors.
- Warnings and low-severity informational messages.

## Nested transactions are a trap

`@@TRANCOUNT` increments with each `BEGIN TRAN`, but only the outermost `COMMIT`
actually commits; an inner `ROLLBACK` rolls back everything. SQL Server has no
true nested transactions. Use savepoints (`SAVE TRAN`) for partial rollback, and
design procedures to detect whether they own the transaction:

```sql
DECLARE @ownTran bit = CASE WHEN @@TRANCOUNT = 0 THEN 1 ELSE 0 END;
IF @ownTran = 1 BEGIN TRAN;
-- ... work ...
IF @ownTran = 1 COMMIT;
```

## Review checklist

1. `SET XACT_ABORT ON` and `SET NOCOUNT ON` present?
2. `CATCH` checks `XACT_STATE()` and rolls back only when appropriate?
3. Error re-raised with `THROW` (not swallowed)?
4. Does the procedure assume it owns the transaction when it might be nested?
5. Any long transaction holding locks across external calls? (Avoid.)
