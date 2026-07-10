# Azure SQL Database / Managed Instance specifics

Advice for Azure SQL differs from on-prem SQL Server in defaults, features, and
operational model. Ground recommendations in the actual target.

## Defaults that differ from on-prem

- **RCSI on by default** — Azure SQL Database enables READ_COMMITTED_SNAPSHOT, so
  readers use row versioning and don't block writers. Do not assume locking READ
  COMMITTED. (See `concurrency-and-isolation.md`.)
- **Query Store on by default** — plan/runtime history is captured; use it for
  regression analysis and forced plans.
- **Automatic tuning** available (auto plan correction, auto index create/drop on
  some tiers) — recommend enabling plan correction; still design for stability.

## Azure SQL Database vs Managed Instance

- **Azure SQL Database** — single database or elastic pool; a subset of the SQL
  Server surface. No cross-database queries by default, no SQL Agent (use elastic
  jobs / Azure Automation), no `USE` to switch databases, `tempdb` managed. DTU
  or vCore purchasing models; serverless auto-pause option.
- **Managed Instance** — near-full SQL Server surface with instance-level
  features: SQL Agent, cross-database queries, CLR, Service Broker, linked
  servers. Choose MI when you need instance features or lift-and-shift fidelity.

## Identity and auth

- Prefer **Microsoft Entra authentication** (managed identities for apps, Entra
  groups for people) over SQL logins; disable SQL auth where policy allows.
- Use **contained users** mapped to Entra principals rather than server logins on
  Azure SQL Database.

## HA / DR and backups (managed)

- Backups and point-in-time restore are automatic; don't draft manual backup
  jobs for Azure SQL DB.
- HA is platform-managed; active geo-replication / failover groups provide DR.
  Design apps for transient-fault retry (connection resiliency) — transient
  errors (e.g. 40197, 40501, 10928 throttling) are expected and should be retried
  with backoff.

## Resource governance

- Service tier caps CPU, memory, IO, and **log write throughput**; a batch that's
  fine on a big box can hit log-rate limits on a small tier. Batch large DML and
  respect `MAXDOP`/resource limits.
- `tempdb` and memory grants are tier-bound — sort/hash spills matter more on
  small tiers.

## SQL Server 2025 features reaching Azure SQL

- Native JSON type, regex functions, and vector type/search are part of the 2025
  wave; availability rolls out across Azure SQL — `[VERIFY]` feature availability
  for the specific service tier/region before relying on it.

## Review checklist

1. Is the target Azure SQL DB or MI? Advice (SQL Agent, cross-db, features)
   depends on it.
2. Did the draft assume locking isolation when RCSI is the default?
3. Auth via Entra managed identity/groups rather than SQL logins?
4. Any manual backup/HA jobs that Azure manages for you? (Remove.)
5. Transient-fault retry assumed in the calling app?
6. Does a large operation risk hitting the tier's log-throughput limit? Batch it.
