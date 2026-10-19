# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project goal

VeloCloud Academy: a personal, single-user, local-only interactive course for learning Arista VeloCloud SD-WAN
from zero to hero (lessons, quizzes, and later a simulator). No login, no accounts; one implicit local profile.

## Stack

- Frontend: React + TypeScript (strict) + Vite + Tailwind CSS v4 + React Router. Lint: oxlint. Format: prettier.
- Backend: Go, `net/http` + chi, pgx for Postgres, plain SQL migrations (embedded, custom runner). No ORM.
- Database: PostgreSQL 16 via docker-compose. Stores progress state only.
- Future simulation engine: Go, inside the backend.

## Layout

```
backend/cmd/server/main.go   entrypoint: config, DB connect, migrate, serve, graceful shutdown
backend/cmd/contentcheck     standalone content validator (make validate-content)
backend/internal/config      env-var config
backend/internal/content     content loader, validator, auto-reloading Provider
backend/internal/grading     pure grading of quiz/review answers (no DB)
backend/internal/review      pure Leitner scheduling, injectable `now`
backend/internal/mastery     pure per-topic accuracy / weak-topic calculation
backend/internal/db          pgx pool + migration runner
backend/internal/store       persistence (SQL lives only here); Leitner updates happen inside its transactions
backend/internal/domain      shared types, no HTTP or SQL
backend/internal/http        chi router and handlers
backend/migrations/          NNNN_name.up.sql / .down.sql, embedded via embed.go
frontend/src/api/            typed API client (all fetch calls go through here)
frontend/src/course/         course context (fetched once from /api/course)
frontend/src/progress/       progress + review-due context (sidebar icons/badges, dashboard)
frontend/src/quiz/           per-type question inputs (questions/), result card
e2e/                         Playwright (TypeScript) regression suite: data-driven from the API, serial, shared DB
frontend/src/mdx/            MDX renderer + component registry (Callout, Diagram, Predict, Term)
frontend/src/mdx/diagram/    diagram engine: canvas (SVG), icons, link-style map, packet animation, step player
frontend/src/glossary/       glossary context (fetched once, shared by <Term> and /glossary)
frontend/src/components/     Layout shell (sidebar, top bar), ErrorPanel
frontend/src/pages/          route pages
content/                     course content files (mounted read-only at /content in the backend)
docs/DECISIONS.md            decision log
```

## Commands

- `make up` / `make down` / `make logs`: start (build + detached, renews the frontend's anonymous node_modules volume), stop, tail logs. App: http://localhost:5173,
  API: http://localhost:8090/api/health (host ports are set in `.env`).
- `make test`: starts the compose `postgres` service, runs `go test ./...` with `TEST_DATABASE_URL` (the Postgres integration tests create and drop a throwaway database each; they skip if the variable is unset, fail if the DB is unreachable), then validates content and type-checks the frontend. `make lint`: go vet, gofmt check, oxlint, prettier check.
  `make fmt`: format everything.
- `make validate-content`: validate `/content` without starting the server (non-zero exit + all errors on failure).
  `make test` also runs it.
- `make e2e`: Playwright regression suite in `e2e/` against the stack on a FRESH database (it runs `docker-compose down -v` first; set `E2E_KEEP_DB=1` to skip the reset). Typecheck it with `cd e2e && npm run typecheck`.
- `make reset-db`: drops the DB volume after a typed confirmation.
- Single Go test: `cd backend && go test ./internal/grading -run TestGrade` (DB tests also need `TEST_DATABASE_URL=postgres://academy:...@localhost:5432/academy?sslmode=disable`).
- Frontend only: `cd frontend && npm run dev | build | typecheck | lint`.
- The backend has no live reload; after Go changes run `make up` again to rebuild. The frontend has HMR.

## Content format (`/content`)

```
content/
  course.yaml                       title, description, sections: [section ids in order]
  sections/<NN-slug>/
    section.yaml                    id, title, summary, estimated_minutes, lessons: [lesson ids in order],
                                    placeholder (optional bool)
    lessons/<NN-slug>.mdx           YAML frontmatter + MDX body
    quiz.yaml                       optional; a top-level list of questions
  diagrams/<id>.yaml                one step-through diagram per file (see below)
  glossary.yaml                     list of glossary terms (see below)
  reference/                        author fact sheets (not course content; see "Reference fact sheets")
  labs/                             reserved for simulator scenarios (not validated yet)
```

- Ids: lowercase letters, digits, single hyphens. Section, lesson and question ids are each unique course-wide;
  the folder/file name prefix (`NN-`) is only for ordering on disk. Order in the app comes from the id lists.
- Lesson frontmatter: `id, title, summary, estimated_minutes` (>0), `topics` (>=1 tag), optional `prerequisites`
  (lesson ids). Do not repeat the title as an `# h1` in the body; the page header already shows it.
- Question: `id, topic, difficulty (1-3), type, prompt (md), options [{id, text, explanation?}], correct, explanation (md),
  lesson_ref?`. `correct` is one id or a list. Types: `single_choice` (exactly 1 correct), `multi_choice` (>=2 correct, not
  all), `true_false` (options optional; `correct: true|false`), `ordering` (`correct` lists every option once, in order).
- Unknown YAML keys are errors (catches typos).
- MDX lessons can use `<Callout>`, `<Diagram id>`, `<Predict>`, `<Term id>`, GFM tables/task lists and fenced code with
  highlighting, without imports (copy-paste examples: `content/README.md`). Register new components in `frontend/src/mdx/registry.ts`.

### Diagrams (`content/diagrams/<id>.yaml`)

`id` (unique), `title`, `alt` (required, for screen readers), `width`/`height` (viewBox units), `nodes[{id,label,kind,x,y,sublabel?,group?}]`,
`groups[{id,label,nodes|box}]`, `links[{id,from,to,style,label?,curve?}]`, `steps[{title,narration(md),highlight[],show[],hide[],packets[],callouts[]}]`.
Zero steps = static diagram. Node kinds: vce vco vcg router l3switch l2switch firewall host cloud internet mpls datacenter saas generic.
Link styles: underlay-internet underlay-mpls underlay-lte overlay-tunnel ipsec control lan; their colors + dash patterns live ONCE in
`frontend/src/mdx/diagram/linkStyles.ts` (Okabe-Ito colorblind-safe palette; also drives the auto legend). Semantics: node and link ids share one namespace;
an element listed in any step's `show` starts hidden and appears at that step (`hide` removes it); non-highlighted elements dim when a step highlights something;
packet `path` is node ids (or link ids between nodes) and must follow links. The backend resolves each packet path to `route {nodes, links}` (served in the API),
so the frontend never re-implements path logic. Under `prefers-reduced-motion` packets are drawn at their destination with no movement. On narrow screens (below ~560px) the SVG keeps a legible minimum width and scrolls sideways inside its own box, with a "swipe" hint, and the view auto-pans to the part of the diagram each step is about; the page itself never scrolls horizontally.

### Reference fact sheets (`content/reference/`)

Research notes that back the lessons, never shown to learners. The backend does not treat the folder as course content (no orphan error) but validates it:
every file except `README.md` must be `.md` and contain a claims table with the header `| claim | source URL | doc version | status |`; each row needs four cells,
status `VERIFIED` (read in an official source), `SECONDARY` (non-official only), `VERIFY` (could not confirm) or `OBSERVED` (seen by the course owner in a real or lab VCO; source must be `user observation YYYY-MM-DD`; files live in `reference/observations/`), and VERIFIED/SECONDARY rows must cite an http(s) URL.
Never invent a source. A lesson sentence must not be stronger than its row; anything not VERIFIED is written cautiously or shown in a warning Callout, and must not be
a quiz answer. `content/reference/README.md` lists the open VERIFY items, the outline-number checks, how each source was read (directly from a PDF, or summarised via WebFetch because Arista's site blocks automated browsers; rows say which), and the per-lesson claim audit for Sections 1 and 2. Fact sheets: routing-overview, static-routes, ospf, bgp, route-redistribution-and-precedence, overlay-flow-control, vco-routing-ui, release-target, wan-background, sdwan-concepts, components, edge-details, orchestrator-details, gateway-details, underlay-overlay, tunnel-types, vcmp-tunnels, ports-protocols, mtu-overhead, dmpo (+ dmpo-monitoring, -steering, -remediation, -load-balancing, -vco-ui), segments, profiles, edge-overrides, business-policy, firewall, applications-qos, vco-config-ui, cloud-vpn, hubs-and-branches, non-sdwan-destinations, network-services-other, overlay-paths, vco-overlay-ui, branding-history, vco-ui-map. Update the sheets before changing a lesson's facts.

### Glossary (`content/glossary.yaml`)

List of `{id, term, aliases?, short (required), long? (md), related? [term ids], lesson_ref?}`. Names and aliases must be unique (case-insensitive) so search is unambiguous.
`<Term id="x" />` (or `<Term id="x">own text</Term>`) renders a popover (hover, focus, tap) with the short definition and a link to `/glossary#term-x`.

### `<Predict prompt choices answer>` 

`answer` is a ZERO-BASED index into `choices`. Explanation = children (Markdown). Deliberately not graded, not stored, never blocks anything; the answer is in the page source.

### Validation (internal/content)

Everything is collected and reported together as `file: message`. Rules: every referenced id exists (course sections,
section lessons, `lesson_ref`, prerequisites); ids unique; no orphan files (section not in course.yaml, lesson not in
section.yaml, any unexpected file under `sections/` or the content root); no prerequisite cycles or self-prerequisites;
required frontmatter and section fields; per-type quiz rules; option ids unique per question. Diagrams: unique ids, required title/alt, known node kinds and link styles,
nodes inside the canvas (identical coordinates rejected), existing node/link ids in links/groups/steps/packets/callouts, packet paths connected through links, steps that reference something.
Glossary: unique ids, unique names/aliases, existing `related` and `lesson_ref`. Reference sheets: claims-table rules above. Lessons: every `<Diagram id>` / `<Term id>` (outside code) must exist, reported with file and line. A file that fails to parse
suppresses "unknown id" errors that it could be the cause of. Startup exits non-zero on invalid content; at runtime the
`Provider` re-checks file mtimes on each request, reloads on change, and the API answers 503 `content_invalid` with
the error list until fixed.

## API

All state is for the single implicit local profile. Unknown ids in DB rows (content changed) are ignored in every response and never deleted.

- `GET /api/health`: service + DB status (503 if DB down).
- `GET /api/course`: title, description, sections with lesson summaries, `placeholder`, `question_count`.
- `GET /api/lessons/{id}`: frontmatter + `source` (raw MDX body, frontmatter stripped).
- `GET /api/diagrams/{id}`: the diagram (nodes, groups, links, steps) with each packet's resolved `route`; 404 unknown id.
- `GET /api/glossary`: `{terms: [...]}` in authored order.
- `GET /api/sections/{id}/quiz`: questions WITHOUT correct answers, explanations or `lesson_ref` (`ordering` options shuffled per request).
- `POST /api/sections/{id}/quiz/attempts` body `{answers:[{question_id, answer}]}`: grades server-side, persists attempt + answers + Leitner
  state in one transaction. Returns `score, total, percentage, passed, results[]` (per question: `correct, partial_score, given,
  correct_answer, explanation, option_explanations` (wrong picks only), `lesson_ref, topic`). 400 for unknown/duplicate question ids or wrong answer shape.
- `GET /api/sections/{id}/quiz/attempts`: history `{id, score, total, created_at}`, newest first.
- `POST /api/sections/{id}/quiz/skip`: records `quiz_skipped_at` (idempotent; first time kept).
- `PUT /api/lessons/{id}/progress` `{status: "visited"|"completed"}`: never downgrades completed; 404 unknown lesson, 400 bad status.
- `GET /api/progress`: per-lesson status, per-section `{lessons_completed, lessons_total, quiz_attempts, best_score_pct, latest_score_pct, quiz_skipped}`, overall %.
- `GET /api/progress/topics`: per-topic `{answers, accuracy_pct, weak, lessons[]}`, weakest first.
- `GET /api/review/queue`: due questions (<=15, most overdue first, no answers) + `due_count, upcoming_count, next_due_at`.
- `POST /api/review/answers`: same grading code and result shape; updates Leitner state; creates NO quiz attempts.
- Content errors: 503 `{error: "content_invalid", details: [{file, message}]}`; unknown ids: 404.

## Schema (migrations 0001, 0002; append-only)

`lesson_progress` (status: not_started|visited|completed), `quiz_attempts`, `quiz_attempt_answers` (per-question result incl. `partial_score`, `given`),
`question_state` (Leitner: box 1..5, counts, `next_due_at`), `review_answers` (review log, feeds topic mastery), `section_state`
(`quiz_skipped_at`, `last_visited_at`). No `user_id` anywhere.

## Grading, review and mastery rules

- single_choice / true_false / ordering: 1 or 0. multi_choice: exact match = 1, else partial = max(0, (correct picked - wrong picked) / total correct);
  `correct` is true only for an exact match. Unanswered = wrong. Wrong JSON shape / unknown or duplicate question id = 400; an unknown *option* id is just wrong.
- Quiz `score` = number of exactly-correct questions; `percentage` = score/total. `passed` (>= 70%) is informational ONLY and must never restrict anything.
- Leitner: new questions start in box 1. Correct: up one box (max 5), next due = now + interval(new box). Wrong: back to box 1, due immediately.
  Intervals: box1 = 1 day, box2 = 3, box3 = 7, box4 = 14, box5 = 30. Every graded quiz or review answer applies this. Review sessions serve at most 15 due questions.
- Topic mastery: last 10 answers (quiz + review) per topic, newest weighs most (weights n..1), using partial scores. Weak = accuracy < 70% with >= 3 answers.
- Soft gating: nothing is ever locked. A failed or skipped quiz never blocks navigation; "Continue" is always enabled; nothing reads `passed` to restrict anything.

## Frontend routes

`/` home, `/glossary` (search by term or alias, `#term-<id>` deep links), `/course/:sectionId`, `/course/:sectionId/:lessonId`, `/course/:sectionId/quiz`, `/progress`, `/review`. Lessons are marked `visited` on open and
`completed` when the end marker scrolls into view (or "Mark as complete"). After a section's last lesson a "Check your understanding" card offers
"Take the quiz" or "Skip for now"; both lead forward. The mobile (<md) sidebar is a drawer behind the "Menu" button.

## Conventions

- Go: standard `gofmt`, `go vet` clean; handlers depend on small interfaces, not on pgx; SQL only in `store`;
  structured logging via `log/slog`; wrap errors with `%w`.
- Migrations are append-only: add a new numbered file, never edit an applied one. Single-user: no `user_id` columns.
- TypeScript strict mode (plus `noUncheckedIndexedAccess`). Call the backend only via `src/api/`.
- Content-in-files rule: course content lives in `/content`, never in the database. The DB holds progress only.
- Soft-gating rule: quizzes never block progress; a failed quiz must not lock any lesson.
- Lessons are compiled from MDX in the browser (`@mdx-js/mdx` `evaluate`); the backend only serves raw source.
- In dev the browser talks to `/api` on the Vite server, which proxies to the backend (no CORS needed).

## Target release

The course targets the **Arista VeloCloud SD-WAN 7.0 documentation set (release train 7.0.x)**: it is the newest set in Arista's documentation library, and Arista Security Advisory 0183
(2026-09-22) lists Orchestrator builds up to 7.0.0.2, so the train is shipping. Its **GA / LTS status is VERIFY**: the 7.0.0 release notes and Arista's "Recommended Releases" article could not be read.
Extra evidence (step 6): the Edge 7x0 datasheet (read directly) says Edges must run software code R7.0.0 for the listed performance, and advisories 0144 (2026-07-27) and 0183 (2026-09-22) list VCO builds 7.0.0.1 and 7.0.0.2. 6.4.0 (GA build R6400-20250429-GA, 2025-05-02, LTS release candidate per its release notes) is the fallback reference. Evidence: `content/reference/release-target.md`; decision: docs/DECISIONS.md.
Facts used in lessons are chosen to hold across 6.4 and 7.0. Confirm the release before writing Section 2.

## Content status

| section | status |
|---|---|
| `introduction` (Section 1) | **Real content, written.** 5 lessons, 3 diagrams (`wan-mpls-backhaul`, `wan-direct-internet`, `vc-components`), 15-question quiz (6 scenarios), 30 glossary terms. Topics: wan-limits, sdwan-benefits, velocloud-components, overlay-and-tunnels, history-naming, vco-orientation (lesson frontmatter topics = quiz topic tags). |
| `architecture` (Section 2) | **Real content, written.** 7 lessons (Edge, Orchestrator, Gateway, underlay/overlay, tunnels, ports and firewall, overhead and MTU), 7 diagrams (`arch-*`), 18-question quiz (7 scenarios, 2 MTU calculations), 20 glossary terms. Quiz topic tags = lesson frontmatter topics: edge-role, orchestrator-role, gateway-role, underlay-overlay, tunnel-architecture, firewall-ports, mtu-overhead. Anything not VERIFIED is hedged or in a warning Callout. |
| `dmpo` (Section 3) | **Real content, written (step 7).** 7 lessons (overview, monitoring, steering, outages and brownouts, remediation, putting it together, reading DMPO in the VCO), 6 diagrams (`dmpo-*`: probes, steering modes, outage failover, brownout FEC, jitter buffer, NACK), 19-question quiz (7 scenarios, 1 ordering, all four types), 22 glossary terms. Quiz topic tags = lesson topics: dmpo-overview, dmpo-monitoring, dmpo-steering, dmpo-outage-brownout, dmpo-remediation, dmpo-scenarios, dmpo-vco-diagnosis. Sources: `content/reference/dmpo*.md` (6 sheets); legacy 2017 white-paper facts are SECONDARY and never a quiz answer. |
| `configuration` (Section 4) | **Real content, written (step 8).** 8 lessons (configuration model, segments, profiles, Edge overrides, business policy, firewall, applications and QoS, reading an Edge's configuration), 5 diagrams (`cfg-*`; `cfg-fw-vs-bp` and `cfg-bp-classify` are labelled illustrative), 19-question quiz (9 scenarios, 1 ordering, all four types), 18 glossary terms. Topics: config-model, config-segments, config-profiles, config-overrides, config-business-policy, config-firewall, config-apps-qos, config-diagnosis. Sources: seven new reference sheets, mostly 7.0 pages via WebFetch; unresolved items in `content/reference/README.md` step 8. |
| `overlay` (Section 5) | **Real content, written (step 9).** 7 lessons (overlay at a glance, Cloud VPN, hubs and spokes, dynamic branch-to-branch, non SD-WAN destinations, other network services, choosing and reading paths), 6 diagrams (`ov-*`), 19-question quiz (8 scenarios, 1 ordering, all four types), 16 glossary terms. Topics: overlay-ways, overlay-cloud-vpn, overlay-hubs, overlay-dynamic, overlay-nsd, overlay-services, overlay-diagnosis. Sources: six new reference sheets (7.0 pages via WebFetch, local Troubleshooting Guide). |
| `sample`, `sample-two` | Dev placeholders, last in `course.yaml`; a later step removes them (tests still use them). |
| `routing` (Section 6) | **Real content, written (step 10).** 8 lessons (big picture, static and default routes, OSPF, BGP, redistribution and route preference, filters and attributes, Overlay Flow Control, reading routes in the VCO), 7 diagrams (`rt-*`; those with prefixes are titled illustrative), 20-question quiz (8 scenarios, 3 ordering, all four types), 34 glossary terms. Topics: rt-model, rt-static, rt-ospf, rt-bgp, rt-redistribution, rt-filters, rt-ofc, rt-diagnosis. Sources: seven new reference sheets (7.0 Troubleshooting Guide read directly; 7.0 and 6.4 pages via WebFetch). Arista uses route preference (lower wins), not administrative distance. |
| Section 7 onward | Not started. |

## Current status

Step 10 complete: housekeeping (Hub wording, Dynamic Branch to Branch default now quoted from the 7.0 Profile page, callout and packet-label placement fixes with unit tests), seven routing fact sheets, the real content of Section 6, and `docs/SIMULATOR-MODEL.md` sections 10 to 15 (routing model, reachability, assumptions 21 to 32, mock routing screens, build plan).
Not built yet: the simulator and mock VCO, labs.
Sources used in step 10: 7.0 Troubleshooting Guide (local PDF, read directly); 7.0 and 6.4 dynamic routing pages, 7.0 Overview, Edge and Profile device settings and Monitor Enterprise through WebFetch (verbatim quotes requested); the dedicated Overlay Flow Control pages were unreadable (HTTP 403).
Open VERIFY items: see `content/reference/README.md` (step 10 list of items needing a real-VCO observation).
