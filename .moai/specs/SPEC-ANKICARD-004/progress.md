# Progress — SPEC-ANKICARD-004

Swift arrow card rendering. Tier M, three plan-phase artifacts plus this file.

## §E.1 Plan-phase Audit-Ready Signal

```yaml
plan_status: audit-ready
plan_complete_at: 2026-10-12
spec_id: SPEC-ANKICARD-004
spec_version: "0.1.4"
tier: M
artifacts: [spec.md, plan.md, acceptance.md, progress.md]
requirements: 14
acceptance_criteria: 15
corpus_rows: 42
depends_on: [SPEC-ANKICARD-003, SPEC-ANKICARD-005]
baseline_head: 7d98ec0
next_audit_iteration: 4
next_audit_scope: delta-only (R3-1, R3-2, R3-3, R3-4, O-2)
next_audit_authorization: granted (decision board d-20261012T131704Z-eab1, scope card:t15 — PASS proceeds to run; FAIL stops, no further rounds)
answers_audit: .moai/reports/plan-audit/SPEC-ANKICARD-004-plan-audit-r3.md
```

Iteration 3 (the last of the three the retry loop allows) returned FAIL 0.75.
v0.1.4 is the targeted fix that the escalation's option 2 describes. It closes
the four blocking findings R3-1 to R3-4 and takes the optional O-2. The
per-finding account is in `spec.md` HISTORY v0.1.4.

A fourth, delta-only audit pass goes beyond the iteration cap. It needs the
user's explicit override; it was **granted** (decision board
`d-20261012T131704Z-eab1`) and ran: plan-audit r4 **PASS 0.92**
(`.moai/reports/plan-audit/SPEC-ANKICARD-004-plan-audit-r4.md`). R4-1 applied
to `plan.md` DD-4 before run. The Tier M PASS threshold is 0.80.

The corpus grows from 38 rows to 42. The four new rows (39-42) were measured
against go-org v1.9.1 from the module cache, in a scratch module outside the
checkout that was deleted afterwards. The baseline is unchanged at `7d98ec0`
(`plan.md` § A.2).

## §E.2 Run-phase Evidence

_<pending run-phase>_

## §E.3 Run-phase Audit-Ready Signal

_<pending run-phase>_

## §E.4 Sync-phase Audit-Ready Signal

_<pending sync-phase>_
