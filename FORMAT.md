# Pointfold Markdown profile, version 0.4

## Model and meaning

A document has a title and an ordered list of root points.
A point has text, optional visual blocks, an optional stable ID, and an ordered list of child points.
No point has a separate summary, body, evidence, or takeaway property.
Containment means “more detail about this point”; the text states the particular relationship.
Children may be reasons, substeps, cases, definitions, qualifications, or inference steps.
Cross-references add navigable links and do not change containment.

“One logical statement per point” is an authoring rule, not a mechanically enforced sentence limit.
Use separate points for independent claims even if punctuation could squeeze them into one sentence.
Do not bury a caveat in a child when doing so makes the visible parent misleading.

## Blocks

Start with one Markdown H1 title, followed only by nested unordered lists.
The title may use inline emphasis or code, but displays as plain text.
Blank lines and ordinary Markdown soft wrapping are allowed.
Use two spaces for a child's indentation; standard CommonMark indentation rules apply.
Bullet markers `-`, `*`, and `+` are accepted.
Ordered lists are rejected because the renderer supplies numbering.

A list item contains one text block followed optionally by visual blocks and child lists.
A second paragraph in the same item is rejected; make it a sibling or child point instead.
An image-only paragraph is a supported visual block rather than extra prose.
Additional headings, quotes, thematic breaks, task-list semantics, footnotes, raw HTML, ordinary code blocks, and link-reference-definition blocks are outside this profile.
Use explicit inline links rather than bare angle-bracket autolinks.

## Optional titles

A short bold phrase can precede the point's statement, as in `**Case for.** The charge could reduce avoidable peak-hour driving.`
This is ordinary Markdown, with no special parser behavior or required title field.
The title adds orientation rather than a second logical claim.
Use it when helpful and leave other points untitled.
A grouping branch may simply be named `**Definitions**`.

## Visual blocks

Attach a GFM-style pipe table, a standalone Markdown image, or a fenced `mermaid` block to the point by indenting it under the bullet after a blank line:

````markdown
- **Comparison.** The policies differ in delay. {#comparison}

  | Policy | Delay |
  | --- | ---: |
  | Immediate | 0 seconds |
  | Fixed | 1 second |

  - These values are illustrative.
- **Flow.** An attempt can succeed or fail.

  ```mermaid
  flowchart LR
    A[Attempt] --> B{Succeeded?}
    B -->|Yes| C[Result]
    B -->|No| D[Failure]
  ```
- The image shows a quadratic curve.

  ![A plot of y equals x squared](assets/parabola.svg)
````

Visual blocks belong to the point itself and receive no separate outline number.
They stay hidden while the point is collapsed, even if it has no child points.
The point's text remains its ordinary text, not a separate summary or body field.
Multiple visual blocks are allowed; they render in source order before the point's child bullets.
An image may also be the whole bullet, as in `- ![Descriptive label](assets/plot.png)`; its alt text supplies the collapsed label.
Tables and Mermaid diagrams require an initial text label for their point.

Images must be local PNG, JPEG, GIF, WebP, SVG, or AVIF files, at most 20 MiB each.
Paths resolve relative to the input Markdown file, or the current directory for stdin.
Percent-encoded spaces are supported.
The renderer embeds the images, so moving the generated HTML does not break them.
For a web image, save it locally and use that path.
Images start at a maximum height of 200 pixels and fit within the available width.
Full size removes those limits, with horizontal scrolling inside the image frame when necessary; Small size restores them.

Diagrams use standard Mermaid syntax in `mermaid` fences, rendered by [Pele](https://pele.run/api) when their branches first open.
Pele 1.13.0 loads from jsDelivr with a pinned version and integrity hash; the CDN load needs an internet connection.
Without JavaScript, tables and images still appear when expanded, and diagram blocks show their source text.
`check` validates placement and references, while diagram syntax errors appear when the point is opened.

## Inline text

Inline code, emphasis, strong emphasis, readable Unicode, and explicit Markdown links are supported.
Inline code and emphasis follow CommonMark behavior.
Soft line breaks are joined into a single point's text.
Do not rely on hard line breaks to separate statements.
This version does not render LaTeX; write `a² = 2b²` or `a^2 = 2*b^2` as inline code.
Non-ASCII natural-language text is supported in points and titles.

External links must be absolute `https://`, `http://`, or `mailto:` links.
Relative file hyperlinks, executable URL schemes, and raw HTML are rejected.
Local image paths are supported only in image syntax and are embedded in the generated artifact.

## Stable IDs

```markdown
- The queue retains failed jobs for another attempt. {#retry-queue}
```

An optional suffix `{#id}` assigns identity to a point.
It must occur at the end of the point text, outside emphasis or inline code, with whitespace before it.
IDs match `[a-z][a-z0-9-]*` and must be unique within a document.
They are case sensitive, hidden in the reader, and optional for unreferenced points.
The suffix is a small extension to Markdown and may remain visible in ordinary Markdown viewers.
Preserve identity across reordering and rewording; do not reuse an old ID for an unrelated statement.

## References and numbering

```markdown
- Retries can increase load on the [retained-job queue](#retry-queue).
```

Fragment links target stable IDs only; forward links are allowed.
Every fragment must resolve, or validation fails.
Do not author links to generated positions such as `#2.1` or `#p-2.1`.
The link renderer appends the target's current outline number to the authored label.

Root points are numbered `[1]`, `[2]`, and so on in source order.
Children append an ordinal, producing `[2.1]`, `[2.2]`, and deeper forms such as `[2.2.1]`.
The H1 title is not numbered.
Moving or inserting points can change these numbers; they identify positions within one document version.
The copy control copies only this positional ID, including brackets.
The separate right-hand `md` control copies the point's Markdown text and any attached visual blocks, excluding its bullet marker, source-only stable-ID suffix, and child bullets.
Inline formatting and original source links are preserved rather than copied as rendered text.
`pointfold refs` bridges displayed numbers to source IDs and line numbers for an agent working on the same version.

## Definitions and citations

There is no glossary or citation syntax beyond ordinary points and links.
An optional definitions branch can place a stable ID on each defined term.
Name that branch `**Definitions**` rather than introducing it with a full sentence.
Define terms naturally in sentences, or use `**prime** - a positive integer greater than one whose only positive divisors are one and itself.`
Authors can link a term to its definition exactly as they link any other point.
Use inline source links near factual claims, or ordinary source points when a shared citation needs a reference target.
Support, opposition, uncertainty, and inference are expressed in text rather than machine-enforced node types.

## Validation boundaries

The CLI checks structural shape, nonempty points, ID syntax and uniqueness, supported links, and resolvable references.
It rejects unsupported block content instead of deliberately hiding it in the reader.
The current maximum nesting depth is 64.
Validation does not establish correct evidence, sound logic, completeness, or useful hierarchy.
The reusable skill governs those authoring decisions.
