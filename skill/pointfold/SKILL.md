---
name: pointfold
description: Create or revise expandable HTML explanations using the Pointfold CLI, with hierarchical Markdown source and addressable points, for research, decisions, discussions, mathematical explanations, algorithms, or codebase notes.
---

# Pointfold explanations

Produce a tree of points that remains useful at different reading depths, using the syntax below.

## Default deliverable: rendered HTML

Using this skill means producing an HTML reader through the Pointfold CLI by default, together with its editable Markdown source.
The user does not need to ask separately for rendering.
An inline bulleted answer alone does not complete the skill's workflow.
Honor an explicit request for Markdown-only, inline text, or another delivery format instead.

Save the outline in the task's designated output directory, then run:

```sh
pointfold check outputs/explanation.md
pointfold render outputs/explanation.md -o outputs/explanation.html
```

Adapt the example paths to the environment's output directory and requested filename.
Fix validation or rendering errors, and confirm that the HTML file was created successfully.
When a preview is available, check that the reader opens with collapsed branches and that a branch expands.
Deliver a link or attachment to the HTML first, with a Markdown source link when useful, and keep the chat reply brief rather than repeating the full outline.

If `pointfold` is missing from PATH, look for an existing binary in a Pointfold checkout or the Go binary directory.
If none is available and Go is installed, install the CLI into the task's scratch directory and invoke that binary directly:

```sh
mkdir -p work/pointfold-bin
GOBIN="$PWD/work/pointfold-bin" go install github.com/Chriscbr/pointfold@latest
work/pointfold-bin/pointfold check outputs/explanation.md
work/pointfold-bin/pointfold render outputs/explanation.md -o outputs/explanation.html
```

Use the environment's scratch directory when it differs from `work/`.
A missing PATH entry alone is not a reason to skip HTML rendering.
If execution, installation, or required environment permissions prevent rendering, explain the concrete blocker and label HTML generation incomplete; provide Markdown as a fallback without claiming it was rendered.

## The unit of writing

Each point contains its own text, optional visual blocks, and optional child points.
A point should be at most one sentence in logical size: one claim, definition, question, instruction, or inference.
Use multiple points for multiple statements, even when a semicolon could join them.
Do not add per-point summary, takeaway, body, evidence, or explanation fields.
Evidence and explanations, when needed, are themselves ordinary points.
An optional short bold title such as `**Case for.**` may precede the statement in the same point.
This title is a reading aid, not another claim, a required field, or special CLI syntax.
Use titles selectively when they help scanning; leave points untitled when a label adds little.
For example, `**Overview.**`, `**Case against.**`, and `**Tradeoffs.**` can help a debate, while a clear pilot proposal may need no label.
`**Pilot proposal.**` is also reasonable when it helps orient the reader.

Write parents that make sense while collapsed.
Children should directly develop their parent by explaining its mechanism, decomposing its scope, justifying an inference, giving a case, or qualifying a claim.
Choose the relationship that suits the material; children do not universally mean support or opposition.
Use siblings for peer statements and children for additional detail about one statement.
Let the words express relationships such as “because,” “however,” “assuming,” and “therefore” when the relationship would otherwise be ambiguous.

## Organize for selective reading

Start with the few broad points needed to understand the subject or follow the procedure.
Use informative statements, optionally preceded by short bold titles, and concise labels for grouping branches.
Allow more levels where they let the reader skip familiar details, without inventing depth or padding a sparse branch.
Stop when another level would merely restate the point above it.
Do not repeat the same statement as an abstract, introduction, and expanded explanation.

Read the document with only its root points visible, then with one more level visible.
At each depth, check that visible statements remain coherent and do not become misleading through hidden qualifications.
Keep assumptions and limits necessary for interpreting a claim in that claim or at a visible peer level.
A collapsed parent must not overstate what its children establish.

## Source syntax

Use one H1 title followed by nested unordered Markdown bullets.
Prefer two spaces per nesting level and one physical line per point; ordinary soft wrapping is also accepted.
Use inline Markdown for emphasis, code, and explicit links.
Do not use additional headings, standalone prose paragraphs, task boxes, raw HTML, or ordinary code blocks in the outline.
For a visual point, put a Markdown table, an image-only paragraph, or a fenced `mermaid` block under its text, separated by a blank line and indented to belong to that bullet.
These blocks appear only when that point is expanded; they are not additional numbered child points or separate prose fields.
Use local image paths relative to the Markdown file so the CLI can embed them in the standalone HTML.
An image-only bullet is also supported, with descriptive alt text supplying the collapsed label.
Images start at a maximum height of 200 pixels and have a Full size control.
Diagrams use Mermaid syntax and are rendered by Pele, loaded client-side from a pinned CDN release.
Write mathematics with readable Unicode or inline code; the initial renderer does not typeset LaTeX.

```markdown
# Why does this search return the first qualifying index?

- The search keeps every possible answer inside `[lo, hi)`. {#invariant}
  - Every earlier value is smaller than the target.
  - Every value at or after `hi` meets the target.
- Updating `hi = mid` preserves the [invariant](#invariant). {#upper-update}
  - Sorted order makes every value from `mid` onward qualify.
  - The index `mid` remains a possible answer.
```

Add a trailing `{#stable-id}` when a point is referenced or needs to retain identity during revision.
IDs start with a lowercase ASCII letter and contain lowercase ASCII letters, digits, or hyphens.
Every ID must be unique in the document.
Link with `[meaningful phrase](#stable-id)` rather than hardcoding an outline number.
Preserve the ID when moving or rewording the same point, and allocate new IDs for genuinely different points.
The renderer hides stable IDs and shows positional references such as `[2.1]`.
Copied numbers refer to the current rendered version, so check the source before interpreting numbers from an older version.

## Use conventions that fit the topic

For unfamiliar or easily conflated terms, consider an early branch named simply `**Definitions**`.
Avoid introductory sentences such as “The definitions below fix the meanings used in the proof.”
Definitions may use the term in a sentence, such as `**Congestion** means delay caused by excess traffic demand.`, or use the form `**prime** - a positive integer greater than one whose only positive divisors are one and itself.`
Give each definition a stable ID and link to it using ordinary cross-references where useful.
Use the defined term consistently, and do not add a glossary when the vocabulary is already clear.

For research, distinguish findings, assumptions, interpretations, and unanswered questions in the point text.
Attach ordinary Markdown source links close to the claims they support.
If a sources branch helps, its entries are ordinary points that can be referenced like any others.
Do the research the task requires; the outline format does not substitute for evidence or permit invented citations.

For a decision or debate, include material counterarguments and the conditions that could change the conclusion.
For a discussion summary, distinguish positions from evidence and avoid treating repetition or votes as proof.
For an algorithm, expose the contract, assumptions, steps, invariants, and termination reasoning where relevant.
For a proof, make inference steps inspectable and cross-reference reused definitions or lemmas.
For a codebase explanation, ground behavior in inspected code and distinguish observed behavior from inferred intent.
These are writing choices rather than required schemas or fixed section templates.

## Revise and hand off

When asked about a rendered reference such as `[3.2]`, find that point in the current Markdown tree.
If available, `pointfold refs document.md` maps current numbers to source lines and stable IDs.
Resolve version ambiguity before silently applying an old positional reference to a different point.
Add explanations under the relevant point and use cross-references when a fact belongs primarily elsewhere.

When revising an explanation, update its Markdown and rerender the HTML through the same default delivery workflow.
Keep HTML readers compact and minimally styled, using browser defaults and collapsed branches initially.
Show authored content and necessary navigation or copy controls, preserve inline Markdown formatting, and omit branding, instructions, counts, and footers.
The right-hand Markdown copy control copies that point's own text and visual source blocks, excluding child bullets, its bullet marker, and its source-only stable-ID suffix.
The CLI validates syntax and references, not sentence size, factual correctness, or quality of the hierarchy.
