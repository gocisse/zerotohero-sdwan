# How to add an observation

You have been in a real or lab VCO and saw something the course could not confirm. This is the exact routine, in order. Nothing here lets an observation be invented: a file is only as good as what you saw, on a date, in a named release.

## Before the lab session

1. `make observations-report`, then open `docs/OBSERVATION-PRIORITIES.md`. The **Top 25** table is your checklist: the items at the top block the most labs, lessons and engine behaviour.
2. For each item you plan to test, write down the fact-sheet row (`<sheet>#<line>`) or the assumption number. You will name it in `Resolves`.
3. Note the release and build the VCO shows (usually in the header or the About dialog). If you cannot find it, you cannot record the observation.

## After the lab session

### 1. Write the observation file

Create `content/reference/observations/<topic>.md` from the template in `content/reference/observations/README.md` (a format sample with fake data is `docs/SAMPLE-OBSERVATION.md`). One `## Observation N` block per thing you saw. Be literal: quote the labels as they appear, describe the screenshot, say what you clicked. Set **Effect** honestly:

- `confirms`: the current content was right.
- `corrects`: right in spirit, wrong in detail (a default, a name, a number).
- `contradicts`: the current content was wrong.

Add a row to the file's claims table for each checkable sentence: source `user observation YYYY-MM-DD`, doc version = the release you saw, status `OBSERVED`.

### 2. Run the validators

```
make validate-content
```

It fails, with every problem listed together, when a block lacks a date, a release, a screen path, what was done or seen, a valid effect or a `Resolves`; when a `Resolves` names a row that does not exist (or matches several) or an assumption that does not exist; and when an OBSERVED row has no observation block of the same date. Fix and re-run until it prints `content OK`.

### 3. Run the report

```
make observations-report
```

The items you resolved now show the file in the **observation** column. The ranking tells you what is next.

### 4. Update the fact sheet row

In the sheet you resolved, change the row's status from `VERIFY` (or `VERIFIED` that you are correcting) to match what you found, and set its source to `user observation YYYY-MM-DD` with the release in the doc-version column. Rewrite the claim text so it states what you observed, no more. If the row is VERIFIED from an official source and you only confirmed it, add the OBSERVED row beside it instead of replacing it.

### 5. Turn the engine assumption into a verified rule

If the observation settles an assumption (`Resolves: assumption N`) or a not-modelled rule:

1. In `backend/internal/sim/rules*.go`, change the rule from `StatusAssumption` (or `StatusNotModelled`) to `StatusVerified`, with `Sheet` and an exact `Claim` substring of the OBSERVED row. `rules_test.go` checks that the row exists and is VERIFIED or OBSERVED.
2. Remove the assumption's entry from `docs/SIMULATOR-MODEL.md` or mark it "settled by observation <file>" (keep the number so old references still resolve).
3. If a switch in the assumptions panel existed only for this assumption, keep it only if the real behaviour is configurable; otherwise remove the switch.
4. Update the trace and screens if they showed a "not modelled" box for it.

### 6. Update lessons, labs and quiz

- **Lessons**: where a lesson hedged this point (warning Callout, "not confirmed"), state the observed fact and drop the hedge, no stronger than your row. Do not add claims beyond the observation.
- **Labs**: if a lab listed the assumption in `relies_on_assumptions`, remove the number once the rule is verified, and fix the symptom text that says the answer rests on it. The lab validator tells you if a dependency is still missing.
- **Quiz**: questions may use an OBSERVED row as a correct answer only once the row is OBSERVED. Add questions only when asked; never turn a hedged point into an answer before the row is OBSERVED.
- **Glossary and diagrams**: search for the term; update hedged wording.

### 7. Run the tests

```
make validate-content && make test && make lint
make e2e
```

## When an observation CONTRADICTS the content

The observation is never edited to fit the course. The lesson, the labs and the engine change instead:

1. Record the observation as it is, with `Effect: contradicts`, and keep the VERIFIED row it contradicts if it came from official documentation: note the release each applies to (the guide may describe another release or a different UI).
2. In the engine, change the behaviour to what you observed (a rule with `StatusVerified` pointing at your OBSERVED row), and fix every place that cited the old rule. `go test ./...` will fail where a test or a lab asserted the old behaviour; update those expectations, not the observation.
3. Labs: re-read each lab that touched the behaviour; the validator re-solves them, so a lab whose answer changed fails until its correct choice and explanation are rewritten.
4. Lessons and quiz: rewrite every sentence and every quiz explanation that stated the old behaviour. Use `docs/OBSERVATION-PRIORITIES.md` to find the lessons and questions that depend on the item.
5. Add a line to `docs/DECISIONS.md` saying what changed and why.
6. Run the checks in step 7.

## What never to do

- Do not record something you did not see, or a guess about why it happens.
- Do not edit an observation after the fact except to fix a typo; add a new dated observation for a new finding.
- Do not paste vendor documentation into an observation; this folder is for what you saw.

## Observing the Edge command line (step 20)

`docs/OBSERVATION-PRIORITIES.md` has a part "debug.py and Edge command line: what I need you to run": one row per command or option found in a source, with exactly what to capture. Run only the read-only rows on a lab Edge you are allowed to log in to, record the Edge release and the time, copy the output unedited (no private keys), and write one observation block per item as above. An observation of a debug.py option resolves the matching VERIFY row of `content/reference/troubleshooting/ts-edge-cli.md`; then update the `debug-script-reference` runbook and the Section 10 Edge command line lesson (and its study guide rating) from the observed text only.

