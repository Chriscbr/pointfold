package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPeleBlocksUseCDNWithoutBundledMermaid(t *testing.T) {
	source := "# Pele\n- A flow. {#flow}\n\n  ```pele\n  flowchart LR\n    A-->B\n  ```\n\n  - A child.\n"
	out, err := parseOutline([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	p := out.ByID["flow"]
	if len(p.Content) != 1 || p.Content[0].Kind != "pele" || !strings.Contains(p.Markdown, "```pele\nflowchart LR") || strings.Contains(p.Markdown, "A child") {
		t.Fatalf("wrong Pele point: %+v", p)
	}
	var page bytes.Buffer
	if err := renderOutline(&page, out, 1, "."); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.String(), `data-renderer="pele"`) || strings.Contains(page.String(), "window.mermaid=") {
		t.Fatal("wrong renderer in Pele-only document")
	}
	head := strings.Split(page.String(), "<body>")[0]
	if !strings.Contains(head, "https://cdn.jsdelivr.net") {
		t.Fatal("Pele CDN is not allowed by the page policy")
	}
	plain, _ := parseOutline([]byte("# Plain\n- A point.\n"))
	page.Reset()
	if err := renderOutline(&page, plain, 1, "."); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Split(page.String(), "<body>")[0], "https://cdn.jsdelivr.net") {
		t.Fatal("plain outlines should not allow external scripts")
	}
}
