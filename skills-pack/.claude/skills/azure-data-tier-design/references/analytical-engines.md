# Analytical Engines — Synapse, Fabric, Azure Data Explorer

## Purpose

The pack scope is OLTP microservices, not analytical / BI workloads. This reference exists for the boundary case: when **a microservice or platform needs an analytical engine alongside the OLTP store** — ad-hoc reporting, long time-range aggregations, dashboard queries that would crush the transactional database. The default answer is **not** to do analytics in the OLTP engine; the question is which analytical engine and how to feed it.

## The architecture pattern

```
OLTP source (Postgres / Cosmos / Mongo / SQL)
  │
  ▼  CDC / change feed / event stream
Lakehouse storage (Delta Lake on ADLS Gen2 / OneLake)
  │
  ▼  Spark notebooks, SQL queries, Power BI
Analytical engine (Fabric Warehouse / Synapse Serverless / Azure Data Explorer)
```

OLTP services keep their own database optimized for transactions. Changes flow into a lakehouse via:

- **Postgres / Azure SQL / MySQL** → Debezium / Azure Database Migration Service / Fabric Mirroring
- **Cosmos DB** → Change feed → Azure Function → Delta tables / Cosmos Mirroring in Fabric
- **Application events** → Service Bus / Event Hubs → Stream Analytics → Delta

The analytical engine queries the lakehouse, not the OLTP source. This isolates transactional performance from analytical load.

## When you actually need an analytical engine

Genuine signals:

- Reporting queries take > 5 seconds on the OLTP database after reasonable indexing
- Time-range queries span months / years and aggregate millions of rows
- BI tools (Power BI, Tableau) connect directly to the database and the load is visible
- Data scientists run ad-hoc SQL / Spark notebooks against production data
- ML feature engineering needs joins across multiple OLTP sources

Not genuine signals:

- "We might want analytics someday" — don't pre-build. Stand it up when the OLTP burden materializes.
- "The CEO asked for a dashboard" — first try a read replica or CQRS read model (`microservices-data-architecture`). The lakehouse is a bigger lift.

## The three analytical engines on Azure

| Engine | Sweet spot | Cost model |
|---|---|---|
| **Microsoft Fabric Warehouse** | SQL analytics over Delta tables in OneLake; modern default | Capacity units (CU) |
| **Microsoft Fabric Lakehouse** (Spark) | Spark notebooks, Delta tables, ML on lakehouse | Capacity units (CU) |
| **Azure Synapse Serverless SQL** | Pay-per-query SQL over data lake files | $5/TB scanned (predictable for ad-hoc) |
| **Azure Synapse Dedicated SQL Pool** | Legacy MPP data warehouse | Provisioned DWU; expensive |
| **Azure Data Explorer (ADX) / Kusto** | Time-series, telemetry, log analytics | Cluster-based |
| **Azure Databricks** | Best-in-class Spark for ML / data engineering | DBUs; separate vendor relationship |

### Microsoft's strategic direction — Fabric

Fabric is Microsoft's converged data platform — Warehouse + Lakehouse + Real-Time Analytics (KQL) + Data Factory + Power BI under one capacity model. **Fabric is the strategic default for new analytical workloads.** Synapse Dedicated SQL Pool is legacy; new work shouldn't land there. Synapse Serverless SQL has been absorbed into Fabric's surface for ad-hoc lakehouse queries.

### Picking between Fabric, Synapse Serverless, ADX

```
Workload                              → Engine
────────────────────────────────────────────────────────────────────
Modern data warehouse, SQL-first      → Fabric Warehouse
Spark notebooks, data science, ML     → Fabric Lakehouse (or Databricks)
Ad-hoc SQL on Parquet/Delta files     → Synapse Serverless SQL or Fabric Warehouse
Time-series telemetry, logs           → Azure Data Explorer / Fabric KQL Database
Existing Synapse Dedicated SQL Pool   → Migration path to Fabric Warehouse
Existing Synapse Pipelines            → Migration path to Fabric Data Factory
```

### Fabric Warehouse — the default for new analytics

Strengths:
- T-SQL surface; familiar for Azure SQL teams
- Native Delta Lake storage in OneLake
- Capacity-based pricing (one Fabric capacity covers Warehouse + Lakehouse + Power BI + KQL)
- Mirroring from Azure SQL / Cosmos / Snowflake — near-zero-ETL replication into the lakehouse

Weaknesses:
- Newer; some maturity gaps vs Synapse Dedicated
- Capacity sizing requires estimation; runaway query can starve other workloads in the same capacity
- T-SQL surface is large but not 100% parity with Azure SQL — verify required features

### Synapse Serverless SQL — when ad-hoc, not warehouse

Strengths:
- Pay-per-query; predictable for low-frequency ad-hoc
- T-SQL over Parquet / Delta / CSV in ADLS

Weaknesses:
- Not for high-frequency dashboards (cost adds up)
- No table caching; every query scans data
- Legacy of the Synapse brand; mind-share migrating to Fabric

### Azure Data Explorer — when time-series

Strengths:
- Optimized for ingestion of high-volume time-stamped data (telemetry, logs, IoT)
- KQL is fast to learn and powerful for log analytics
- Used by Azure Monitor, Sentinel, and Defender — your existing dashboards run on it

Weaknesses:
- Niche for general analytics; not a SQL warehouse
- Cluster-based; more ops than serverless options

## Don't do analytics in the OLTP engine

The anti-pattern: business asks for a "small report"; engineering adds a complex aggregation query to Postgres / Cosmos; over time, more reports stack; eventually the OLTP database is fighting analytical load on top of transactional load. Common detection signals:

- `pg_stat_statements` top-N includes queries with `GROUP BY` over millions of rows
- Cosmos cross-partition queries dominate RU consumption
- Azure SQL Query Store shows reporting queries in the top-10 by total time
- The database periodically slows down during business hours but not after-hours

Fix: extract reports to a lakehouse + analytical engine. Use the CDC / mirror path so OLTP performance is isolated.

## Don't do OLTP in the analytical engine

Reverse anti-pattern: someone thinks Fabric Warehouse "is fast and we have it, let's just use it for the OLTP service too." Analytical engines have:

- High query startup overhead (seconds to first row)
- Limited or no transactional semantics across rows
- Append-optimized storage; updates are expensive
- Higher per-query cost; not designed for thousands of small reads/sec

The OLTP database stays Postgres / Cosmos. The analytical engine lives downstream.

## Feeding the lakehouse — CDC and mirroring

### Postgres → Lakehouse

- **Fabric Mirroring for Postgres** (preview/GA depending on date) — managed CDC replication
- **Debezium + Event Hubs + Stream Analytics + Delta sink** — open-source CDC
- **Azure Database Migration Service** — periodic batch (not real-time)

### Azure SQL → Lakehouse

- **Fabric Mirroring for Azure SQL** — managed CDC, near-zero-ETL
- **Change Tracking / Change Data Capture (CDC)** in SQL Server with a custom pipeline

### Cosmos DB → Lakehouse

- **Cosmos Mirroring in Fabric** — near-zero-ETL
- **Cosmos analytical store** (Synapse Link) — column-store sidecar on the same Cosmos data
- **Change feed → Azure Function → Delta sink** — custom pipeline

### Application events → Lakehouse

- **Event Hubs → Stream Analytics → Delta** — for high-volume streaming
- **Service Bus → consumer → Delta** — for moderate volume with ordering

## Cost shape

| Workload | Engine | Order-of-magnitude monthly |
|---|---|---|
| 100GB warehouse, 50 queries/day | Synapse Serverless SQL | ~$50–200 |
| 1TB warehouse, dashboard always on | Fabric F2 / F4 capacity | ~$300–600 |
| 10TB warehouse, heavy use | Fabric F32 / F64 | ~$3K–8K |
| 1TB time-series, telemetry | Azure Data Explorer | ~$500–2K depending on cluster |
| Legacy 1TB Synapse Dedicated DW100c | Synapse Dedicated | ~$1.5K |

Verify with the current Azure pricing calculator. The point: analytical engines are not cheap; right-size by measured demand, not "future-proofing."

## Anti-patterns

- **Reporting queries in OLTP.** Move them to a lakehouse before they degrade transactional performance.
- **OLTP in analytical engine.** Wrong tool; latency and cost shape are punitive.
- **Going to Synapse Dedicated SQL Pool for new work.** Legacy. Fabric Warehouse instead.
- **Direct Power BI connection to Postgres / Azure SQL prod.** Risk of dashboard queries slowing transactional traffic. Use a lakehouse or a read replica.
- **Stand up Fabric Warehouse before naming what queries will run on it.** Like Cosmos partition keys, you can over-spec the capacity and discover the queries don't match.
- **No mirroring / CDC plan.** Standing up a lakehouse with no automated source-to-lake pipeline = stale data, manual loads, mistrust of the warehouse.

## Verification questions

1. Has the actual OLTP burden been measured (slow analytical queries showing up in Query Store / `pg_stat_statements`) before adding an analytical engine?
2. Is the source-to-lakehouse CDC path defined (Fabric Mirroring, Debezium, change feed)?
3. For Fabric: has capacity been sized against measured query volume, not a guess?
4. Is the analytical engine clearly separated from OLTP — Power BI / dashboards don't connect to the transactional database?
5. For Synapse Dedicated SQL Pool: is there a migration plan to Fabric, or is this strictly maintenance?
6. Is the cost trajectory tracked monthly (Fabric capacity utilization, Synapse Serverless query volume)?

## What to read next

- `engine-selection.md` — analytical row in the decision table
- `microservices-data-architecture` skill — CQRS read model as the lighter-weight alternative to a full lakehouse
- `data-migration-patterns.md` — CDC patterns from OLTP to lakehouse
- `azure-microservices-cost-review` skill — capacity sizing economics
- `azure-microservices-observability` skill — separating OLTP and analytical query dashboards
