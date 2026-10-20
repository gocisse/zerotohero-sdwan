# Course content

All course material lives here as files; the database stores only progress state.
The folder is mounted read-only into the backend container at `/content`.

Format (validated at startup and by `make validate-content`; full spec in `CLAUDE.md`):

```
content/
  course.yaml                  # title, description, ordered section ids
  sections/<NN-slug>/
    section.yaml               # id, title, summary, estimated_minutes, ordered lesson ids
    lessons/<NN-slug>.mdx      # YAML frontmatter + MDX body
    quiz.yaml                  # section quiz questions
  diagrams/<id>.yaml           # one step-through network diagram per file
  glossary.yaml                # glossary terms (each with guide_topics or beyond_guide: true)
  study-guide.yaml             # the study guide: sections G1..G5, topics and what the course offers for each
  reference/                   # author fact sheets with sourced claims tables (see reference/README.md)
  topologies/                  # simulator teaching topologies (strict YAML, validated; see docs/SIMULATOR-MODEL.md section 16)
  labs/                        # simulator scenarios (later step)
```

Section, lesson and question IDs are stable slugs; they are the keys used in `lesson_progress.lesson_id`
and `quiz_attempts.section_id`.

## Teaching components (usable in any lesson, no imports)

Everything is validated when the backend starts (and by `make validate-content`): unknown diagram or term ids in a
lesson are reported with the lesson file and line.

### Callout

```mdx
<Callout variant="gotcha" title="Watch out">
  Variants: `info`, `warning`, `tip`, `gotcha`. **Markdown** works inside.
</Callout>
```

### Diagram

```mdx
<Diagram id="rt-lan-to-lan" />
```

The id is the `id:` of a file in `content/diagrams/`. A diagram with `steps` gets a player (Previous / Next / Play / Replay,
left/right arrow keys); a diagram without steps is a static picture.

```yaml
# content/diagrams/my-diagram.yaml
id: my-diagram                       # unique across all diagrams
title: My diagram
alt: Plain-text description for screen readers.
width: 900                           # canvas (viewBox units); node x/y must be inside it
height: 520
groups:                              # optional site/region boundaries
  - { id: site-a, label: Site A, nodes: [host-a, edge-a] }   # or: box: {x, y, width, height}
nodes:
  - { id: host-a, label: Host A, kind: host, x: 70, y: 110, sublabel: optional }
  - { id: edge-a, label: Edge A, kind: vce, x: 210, y: 110 }
links:
  - { id: lan-a, from: host-a, to: edge-a, style: lan, label: optional, curve: 0 }
steps:                               # optional; omit for a static diagram
  - title: Short title
    narration: Markdown **text** shown under the diagram.
    highlight: [host-a, lan-a]       # node/link ids to emphasize; everything else dims
    show: [edge-a]                   # elements listed in any step's `show` start hidden and appear here
    hide: []                         # remove elements again
    packets:                         # animate along a path (node ids, or link ids between nodes)
      - { path: [host-a, edge-a], label: pkt, color_style: lan, delay_ms: 0 }
    callouts:
      - { target: edge-a, text: A speech bubble. }
```

- Node `kind`: `vce vco vcg router l3switch l2switch firewall host cloud internet mpls datacenter saas generic`.
- Link `style`: `underlay-internet underlay-mpls underlay-lte overlay-tunnel ipsec control lan`. Each has its own color
  AND dash/weight; the legend lists only the styles a diagram uses.
- Node and link ids share one namespace (steps refer to both). A step must reference something. A packet path must follow links
  (the validator rejects a path that skips a link). `curve` bends a link (positive/negative picks the side).
- Under `prefers-reduced-motion`, packets are drawn at their destination with no movement.

### Predict

```mdx
<Predict
  prompt="Which of these is a **fruit**?"
  choices={["Carrot", "Apple", "Potato"]}
  answer={1}
>
  Explanation in **Markdown**, revealed after the reader answers or taps "I'm not sure".
</Predict>
```

`answer` is the ZERO-BASED index into `choices`. It is not graded or stored and never blocks anything; the answer is visible in the page
source, so it is for thinking, not scoring.

### Term

```mdx
A <Term id="route-preference" /> in a sentence, or <Term id="overlay-flow-control">with your own wording</Term>.
```

Shows a popover with the short definition and a link to the glossary entry (hover, keyboard focus, or tap).

```yaml
# content/glossary.yaml
- id: route-preference            # unique
  term: Route preference
  aliases: [preference]                 # optional; the glossary search also matches these
  short: One-sentence definition (required).
  long: Optional **Markdown** longer definition.
  related: [overlay-flow-control]    # optional term ids
  lesson_ref: rt-redistribution  # optional lesson id: "Where this is taught"
```

## Study guide (`study-guide.yaml`) and glossary `guide_topics`

The learner studies from a six-section guide; this file maps it onto the course. It is a navigation layer: titles are labels, nothing from the guide is course content.

```yaml
beyond:                       # the glossary group for terms that belong to no guide topic
  title: Beyond the study guide
  reason: Why those terms have no topic.
unmapped_lessons: [ov-glance] # lessons that deliberately belong to no topic
sections:
  - id: G1
    title: Arista VeloCloud SD-WAN
    topics:
      - id: why-traditional-wan            # lowercase, unique across the whole file
        title: Why Traditional WAN Architectures Are No Longer Enough
        kind: concept                      # concept | config | topology
        coverage: full                     # full | partial | none
        note: One sentence: what is and is not covered.
        lessons: [intro-why-wans-fall-short]
        quiz_topics: [wan-limits]          # quiz question topic tags
        diagrams: [wan-mpls-backhaul]
        labs: [mtu-ipsec]                  # never self-check-*
        sim_screens: [map]                 # see SimScreens in backend/internal/content/studyguide.go
```

Rating rules (be conservative; gaps stay visible):

- `full`: a lesson really teaches the topic. Needs at least one lesson.
- `partial`: a lesson teaches part of it, or the concept and where it lives in the VCO but not a step-by-step settings walkthrough (Config topics are partial at best). Needs at least one lesson.
- `none`: nothing in the course teaches it. Must have no lessons. The dashboard shows it as "Not covered in this course yet" and does not count it in progress.
- Update the `note` whenever the rating changes. Do not raise a rating without a lesson that earns it.

Run `make validate-content`: it reports every problem at once (unknown ids, `none` with a lesson, `full` without one, unknown glossary topics). A lesson that is neither mapped nor listed under `unmapped_lessons` is a warning. `make course-map` renders the mapping into `docs/COURSE-MAP.md`.

In `glossary.yaml`, give each term `guide_topics: [topic-id, ...]` (primary topic first: the one the term's lesson teaches, and where a `#term-<id>` deep link opens it) or `beyond_guide: true`. A term with several topics is stored once and shown under each. Never edit a definition to place it.

## Topology lessons (Section 7) and observation candidates

Section 7 (`sections/07-topologies`) follows the same lesson rules as the others and adds three conventions:

- **Tags.** A figure or sentence that is not plainly a documented statement is tagged `(datasheet)`, `(documented)` or `(design practice)`. Design practice is our reasoning and is never written as a vendor statement. Bronze, Silver and Gold are vendor terms (`reference/bronze-silver-gold.md`); "option 1 and 2" is the course's word for the guide's variants.
- **The box.** Every Section 7 lesson has a tip Callout titled `Check this in your own VCO or site` just before the final bridge, listing 3 to 5 bullets in this exact shape (a test checks them):
  `- **[observation candidate]** Open: <screen>. Expect: <what the documentation says you will see>. Docs may differ: <where it may not match>.`
  A line without `Open:` (for example "In a lab only: ...") is allowed; `Expect:` and `Docs may differ:` are required. Avoid `{` and `<` in these lines (MDX).
- **They are not observations.** `make observations-report` lists them per lesson under "Candidates from topology lessons" in `docs/OBSERVATION-PRIORITIES.md`. To turn one into an OBSERVED row, follow `docs/HOW-TO-ADD-AN-OBSERVATION.md`; nothing is ever written to `reference/observations/` by the course author.

## Network services configuration lessons (Section 8) and settings sheets

- **Settings sheets** (`reference/config/`, one per service): the claims table is the settings table. A field is one row that starts `Field "<name>" (applied: ...)`; give allowed values, and a default only if the documentation states one. Menu paths, steps, limits, prerequisites, what is shown afterwards, 6.4 versus 7.0 differences and open items are rows with the prefixes `Menu path:`, `Steps:`, `Limit:`, `Prerequisite:`, `Shown afterwards:`, `6.4 vs 7.0:` and `Needs observation:` (status VERIFY).
- **Lessons.** Each has a `## Settings reference` section (tables may sit under `###` sub-headings) with the field name in the first column; the name must occur in a VERIFIED or OBSERVED row of a `config/` sheet. Fields the pages did not give belong in a warning Callout, not in the table. Avoid `{`, `<` and the less-than character in text (MDX); write "a less-than sign".
- The "Check this in your own VCO" box uses the same `[observation candidate]` line format as Section 7.

## Segment, Profile and policy configuration lessons (Section 9)

Section 9 follows the Section 8 rules (settings sheets in `reference/config/`, a `## Settings reference` section whose field names must exist in VERIFIED rows, the "Check this in your own VCO" box). Extra conventions:

- **Worked examples** use documented fields and allowed values only; the values you choose (an application name, a VLAN) are said to be illustrative.
- **Diagrams** that show behaviour the pages do not describe (a Profile change reaching an overriding Edge, a firewall verdict) are titled illustrative and end by naming what is not documented.
- **Simulator links** (`[Try it in the simulator](/sim/...)`) are used only where the mock screen really shows the setting, and the lesson says what the mock does not model. A data-driven e2e test opens every simulator link of the section.

