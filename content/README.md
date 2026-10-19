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
  glossary.yaml                # glossary terms
  reference/                   # author fact sheets with sourced claims tables (see reference/README.md)
  labs/                        # simulator scenarios (later step)
```

Section, lesson and question IDs are stable slugs; they are the keys used in `lesson_progress.lesson_id`
and `quiz_attempts.section_id`. `sections/00-sample` is placeholder content used to exercise the engine.

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
<Diagram id="sample-branch-hub" />
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
A <Term id="placeholder-term" /> in a sentence, or <Term id="placeholder-alpha">with your own wording</Term>.
```

Shows a popover with the short definition and a link to the glossary entry (hover, keyboard focus, or tap).

```yaml
# content/glossary.yaml
- id: placeholder-term            # unique
  term: Placeholder term
  aliases: [PT]                   # optional; the glossary search also matches these
  short: One-sentence definition (required).
  long: Optional **Markdown** longer definition.
  related: [placeholder-alpha]    # optional term ids
  lesson_ref: sample-markdown     # optional lesson id: "Where this is taught"
```
