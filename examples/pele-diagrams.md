# Render Mermaid syntax with Pele

- **Flow.** A point stays compact until the reader opens it. {#flow}

  ```pele
  flowchart LR
    A[Markdown outline] --> B[Pointfold CLI]
    B --> C[Local HTML]
    C --> D{Open a point}
    D --> E[Read its details]
  ```

  - The `pele` fence selects Pele rather than the bundled Mermaid renderer.
- **Sequence.** Pele loads from a pinned CDN release when a diagram first opens.

  ```pele
  sequenceDiagram
    participant Reader
    participant Page
    participant CDN
    Reader->>Page: Expand a diagram point
    Page->>CDN: Load Pele
    CDN-->>Page: Library
    Page-->>Reader: Rendered SVG
  ```

  - Later diagrams reuse the loaded library.
- Pele blocks use [Mermaid syntax](https://pele.run/api) and need an internet connection for the CDN load.
  - Regular `mermaid` blocks remain available for fully offline diagrams.
