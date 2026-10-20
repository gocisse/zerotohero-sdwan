# Observations

This folder holds **user observations**: facts seen directly in a real or lab VCO and recorded by the course owner. They get the status **OBSERVED**.
Nobody else writes here, and nothing is invented: the folder holds this template only until you add a file.
The step-by-step routine is in `docs/HOW-TO-ADD-AN-OBSERVATION.md`; a sample with obviously fake data is `docs/SAMPLE-OBSERVATION.md`.
Your lab checklist, ranked, is `docs/OBSERVATION-PRIORITIES.md` (`make observations-report`).

## Status OBSERVED

- **OBSERVED** means: seen on screen (or in a packet capture, CLI or log) in a named release on a named date, and described in your own words, with an optional screenshot description.
- It outranks VERIFY and SECONDARY for lessons and quiz answers, and sits beside VERIFIED. If an observation contradicts a VERIFIED row, record both and say which release each applies to.
- The source column of an OBSERVED row must read exactly `user observation YYYY-MM-DD` (URLs are not accepted for OBSERVED).
- The doc-version column states the release observed, for example `7.0.0.2 (lab)`.

## Template for one observation

One file per topic (for example `cloud-vpn-lab.md`). A file holds a header, one `## Observation N: <title>` block per thing you saw, and the claims table of the OBSERVED rows that result. Every field is required; the validator fails the build when one is missing.

```
# <topic> observations

Environment: <hosted or lab, UI new or classic, anything about the setup>. Observed by: <you>.

## Observation 1: <short title>

- Date: 2026-10-20
- Release and build: <exactly as the VCO shows it, for example 7.0.0.2 build R7002-...>
- Screen / menu path: <where you were, with the exact menu names on screen>
- What was done: <the steps you took, so someone could repeat them>
- What was seen: <the text on screen, or a description of the screenshot; quote labels exactly>
- Resolves: <fact-sheet name>: <an excerpt of the row, unique within that sheet>
- Resolves: assumption <N>
- Effect: confirms | corrects | contradicts

| claim | source URL | doc version | status |
|---|---|---|---|
| What you saw, in one checkable sentence. | user observation 2026-10-20 | 7.0.0.2 (lab) | OBSERVED |
```

- `Resolves` may be repeated; each line names a fact-sheet row (`<sheet file name without .md>: <excerpt>`; the excerpt must match exactly one row of that sheet) or an assumption number listed in `docs/SIMULATOR-MODEL.md`.
- `Effect`: **confirms** (the content was right), **corrects** (right in spirit, wrong in detail), **contradicts** (the content was wrong). A correcting or contradicting observation means the lessons, labs and engine change, never the observation. See the how-to.
- `Date` is YYYY-MM-DD and must equal the date in the OBSERVED row's source column.

## What `make validate-content` checks here

- Every observation file has at least one observation block; every block has a real date, a release, a screen path, what was done, what was seen, an effect from the list, and at least one `Resolves`.
- Every `Resolves` names a row that exists exactly once, or an assumption number that exists.
- Every OBSERVED row anywhere in `content/reference/` is supported by an observation block dated like the row (in its own file, or naming it in `Resolves`), and so is every engine rule that rests on an OBSERVED row.

## Which items an observation could settle

`docs/OBSERVATION-PRIORITIES.md` ranks every VERIFY row and assumption by how much content depends on it. The running list of open items is also in `../README.md`.
