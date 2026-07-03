# Improvement backlog — Leadership Status Agent

| Priority | Issue | Why it matters | Required fix | Required before |
|---|---|---|---|---|
| P0 | F-001 evals never run | blocks production readiness (firewall) | run the eval plan; read transcripts | production |
| P0 | F-002 unscoped Bash | excess agency / tool-misuse risk | drop Bash or sandbox it | pilot |
| P1 | F-003 advisory HITL | leader could get an unreviewed deck | make pre-leader review an enforced gate | production |
| P1 | F-004 no monitoring/rollback | can't detect/undo a bad run | add monitoring + rollback + max-turn cap | production |
