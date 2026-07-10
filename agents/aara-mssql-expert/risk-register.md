# Risk Register — aara-mssql-expert

## Risk Table

| ID | Risk | Likelihood | Impact | Mitigation | Owner | Status |
|---|---|---|---|---|---|---|
| R-001 | Hallucinated column names, types, or index definitions if schema is not verified from source | [TODO] | [TODO] | [TODO] | Raja Shekar Bollam (acting engineering lead) | open |
| R-002 | Dynamic SQL vulnerable to injection if not parameterized via sp_executesql | [TODO] | [TODO] | [TODO] | Raja Shekar Bollam (acting engineering lead) | open |
| R-003 | Parameter-sniffing plan regressions if plan-stability advice is wrong | [TODO] | [TODO] | [TODO] | Raja Shekar Bollam (acting engineering lead) | open |
| R-004 | Incorrect isolation assumptions (Azure SQL DB defaults to RCSI, unlike on-prem) | [TODO] | [TODO] | [TODO] | Raja Shekar Bollam (acting engineering lead) | open |
| R-005 | Client schema definitions must never leak across engagements | [TODO] | [TODO] | [TODO] | Raja Shekar Bollam (acting engineering lead) | open |

Risks carry forward from intake; the architect adds likelihood, impact, and mitigation during blueprint review.
