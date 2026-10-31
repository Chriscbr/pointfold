# Mermaid diagrams

- **Flow.** A point stays compact until the reader opens it. {#flow}

  ```mermaid
  flowchart LR
    A[Markdown outline] --> B[Pointfold CLI]
    B --> C[HTML file]
    C --> D{Open a point}
    D --> E[Read its details]
  ```

  - The diagram and child points appear on expansion.
- **Sequence.** The page loads the diagram renderer when needed.

  ```mermaid
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
- **Class.** Each point can contain other points.

  ```mermaid
  classDiagram
    class Point {
      text
      stableID
    }
    Point "1" *-- "0..*" Point : children
  ```
- **Mindmap.** An outline can explain several kinds of material.

  ```mermaid
  mindmap
    root((Pointfold))
      Decisions
      Proofs
      Algorithms
      Codebase notes
  ```
