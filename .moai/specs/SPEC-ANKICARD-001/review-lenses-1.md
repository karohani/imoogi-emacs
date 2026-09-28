# Review lenses — round 1 (2026-09-05)

Only 1 of 4 read-only lenses completed; the other three (requirement testability/GEARS, scope coherence/parent reconciliation, AC/command validity) were terminated by API rate limit (HTTP 429, session limit, resets 15:00 KST) before producing findings.

## Lens: frontmatter schema conformance — completed
- Verdict: no BLOCKING, no MAJOR. 12/12 required fields present, canonical order, correct types; `id` regex PASS; `phase` release label; `tags` comma string; no rejected aliases; HISTORY placed immediately after frontmatter; `tier: L` and `amendment_of` schema-sanctioned.
- MINOR F14: `related_specs` not in schema § Optional Fields table (sanctioned by manager-spec.md:151; no lint consequence). Remedy schema-side.
- MINOR F15: `### Amendments` vs schema's literal `## Amendments` — schema self-inconsistent ("sub-section"); grep still matches. No change required.
- MINOR F16: `prior_completed_sha` recorded as unavailable (parent out-of-repo, frontmatter `in-progress`). Addressed, not satisfied.
- MINOR F17: `amendment_of` row presupposes a completed parent; parent is `in-progress`. Self-disclosed at spec.md:61-64.

## Pending for revision (from orchestrator verification)
- REQ budget: spec.md carries 42 REQ vs Tier L ceiling 25 (spec-workflow.md:148-152). Consolidate to ≤25 without dropping obligations; §3.8 non-goals → out-of-scope prose; AC set (24) stays ≤25.
