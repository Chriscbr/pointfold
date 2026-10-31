package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVisualBlocksAndCopyMarkdown(t *testing.T) {
	source := "# Visuals\n- **Definitions**\n  - A term. {#term}\n- **Comparison.** See [term](#term). {#table}\n\n  | A | B |\n  | --- | ---: |\n  | **bold** | 2 |\n\n  - A child that must not be copied with its parent.\n- A diagram.\n\n  ```mermaid\n  graph TD\n    A-->B\n  ```\n"
	out, err := parseOutline([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	point := out.ByID["table"]
	if len(point.Content) != 1 || point.Content[0].Kind != "table" || len(point.Children) != 1 {
		t.Fatalf("wrong point structure: %+v", point)
	}
	want := "**Comparison.** See [term](#term).\n\n| A | B |\n| --- | ---: |\n| **bold** | 2 |"
	if point.Markdown != want {
		t.Fatalf("copied Markdown:\n%s\nwant:\n%s", point.Markdown, want)
	}
	if strings.Contains(point.Markdown, "must not be copied") || strings.Contains(point.Markdown, "{#table}") {
		t.Fatal("point copy includes child or source metadata")
	}
	var page bytes.Buffer
	if err := renderOutline(&page, out, 1, "."); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<table>", "<strong>bold</strong>", `class="diagram"`, `window.pele.mount`, `data-markdown="**Comparison.** See [term](#term).`} {
		if !strings.Contains(page.String(), want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(page.String(), "<details open>") {
		t.Fatal("visual points must begin collapsed")
	}
}

func TestHeaderOnlyTable(t *testing.T) {
	out, err := parseOutline([]byte("# Empty table\n- A table.\n\n  | A | B |\n  | --- | --- |\n\n- Next.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Points[0].Content[0].Markdown; got != "| A | B |\n| --- | --- |" {
		t.Fatalf("wrong table source: %q", got)
	}
}

func TestImagesAreEmbeddedRelativeToInput(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "assets"), 0755)
	os.WriteFile(filepath.Join(dir, "assets", "test image.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1000" height="640"><rect width="1000" height="640"/></svg>`), 0644)
	source := "# Images\n- A labeled image. {#image}\n\n  ![Alt text](assets/test%20image.svg)\n\n  - A child.\n- ![Image-only alt](assets/test%20image.svg) {#standalone}\n"
	out, err := parseOutline([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if out.ByID["standalone"].Markdown != "![Image-only alt](assets/test%20image.svg)" || len(out.ByID["image"].Content) != 1 {
		t.Fatalf("wrong image points: %+v", out.Points)
	}
	var page bytes.Buffer
	if err := renderOutline(&page, out, 1, dir); err != nil {
		t.Fatal(err)
	}
	if strings.Count(page.String(), `src="data:image/svg+xml;base64,`) != 2 {
		t.Fatal("images were not embedded")
	}
	os.Remove(filepath.Join(dir, "assets", "test image.svg"))
	if err := renderOutline(&bytes.Buffer{}, out, 1, dir); err == nil {
		t.Fatal("missing image must fail rendering")
	}
}

func TestVisualReferencesAndInvalidPlacement(t *testing.T) {
	for _, source := range []string{
		"# Broken\n- Table.\n\n  | A |\n  | --- |\n  | [missing](#missing) |\n",
		"# Broken\n- Inline ![image](x.png) text.\n",
		"# Broken\n- ![](x.png)\n",
		"# Broken\n- A block.\n\n  ![x](javascript:alert%281%29)\n",
	} {
		if _, err := parseOutline([]byte(source)); err == nil {
			t.Fatalf("accepted invalid visual source: %s", source)
		}
	}
}
