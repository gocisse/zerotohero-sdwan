# Observations

This folder holds **user observations**: facts seen directly in a real or lab VCO and recorded by the course owner. They get the status **OBSERVED**.
Nobody else writes here, and nothing is invented: the folder starts with this template only.

## Status OBSERVED

- **OBSERVED** means: seen on screen (or in a packet capture, CLI or log) in a named release on a named date, and described in your own words, with an optional screenshot description.
- It outranks VERIFY and SECONDARY for lessons and quiz answers, and sits beside VERIFIED. If an observation contradicts a VERIFIED row, record both and say which release each applies to.
- The source column must read exactly `user observation YYYY-MM-DD` (the validator enforces this; URLs are not accepted for OBSERVED).
- The doc-version column states the release observed, for example `7.0.0.2 (lab)`.

## How to add one

Create `content/reference/observations/<topic>.md` (for example `cloud-vpn-lab.md`) with the same claims table as the other sheets:

```
# <topic> observations

Environment: <release, build, hosted or lab, UI new or classic>. Observed by: <you>. Notes: <anything about the setup>.

| claim | source URL | doc version | status |
|---|---|---|---|
| What you saw, in one checkable sentence. Screenshot: <describe what it showed>. | user observation 2026-10-20 | 7.0.0.2 (lab) | OBSERVED |
```

`make validate-content` checks every file in this folder (README.md excepted).

## Which VERIFY items an observation could settle

The fact sheets say so in a "needs observation" note on the row. The running list is in `../README.md`, under "Open VERIFY items that need a real-VCO observation".
