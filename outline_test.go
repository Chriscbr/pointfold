package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIdentitySurvivesReordering(t *testing.T) {
	sources := []string{
		"# Decision\n- See [the constraint](#constraint).\n- The constraint applies. {#constraint}\n  - A narrower case applies too.\n",
		"# Decision\n- The constraint applies. {#constraint}\n  - A narrower case applies too.\n- See [the constraint](#constraint).\n",
	}
	for i, source := range sources {
		out, err := parseOutline([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		expect := []string{"2", "1"}[i]
		if out.ByID["constraint"].Number != expect {
			t.Fatalf("wrong target after reordering: %+v", out.ByID["constraint"])
		}
		var page bytes.Buffer
		if err := renderOutline(&page, out, 1, "."); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(page.String(), `class="xref" href="#p-`+expect+`"`) {
			t.Fatalf("cross-reference did not follow the target: %s", page.String())
		}
		if strings.Contains(page.String(), "{#constraint}") {
			t.Fatal("source identity leaked into visible text")
		}
	}
}

func TestInvalidInput(t *testing.T) {
	tests := []struct{ name, source, want string }{
		{"missing title", "- A point.\n", "add a title"},
		{"empty", "# Empty\n", "at least one"},
		{"empty point", "# Empty\n-\n", "own text"},
		{"only ID", "# Empty\n- {#empty}\n", "text before"},
		{"prose", "# Title\n\nA paragraph.\n", "expected a nested bullet"},
		{"heading", "# Title\n- Point.\n## Section\n", "one H1"},
		{"multiple paragraphs", "# Title\n- First.\n\n  Second.\n", "one text block"},
		{"ordered", "# Title\n1. Point.\n", "unordered bullets"},
		{"duplicate", "# Title\n- First. {#x}\n- Second. {#x}\n", "duplicate ID"},
		{"bad ID", "# Title\n- Point. {#Upper}\n", "invalid stable ID"},
		{"missing ref", "# Title\n- See [missing](#missing).\n", "unresolved cross-reference"},
		{"numeric ref", "# Title\n- See [point](#1.2).\n", "unresolved cross-reference"},
		{"script", "# Title\n- Text <script>alert(1)</script>\n", "raw HTML"},
		{"remote image", "# Title\n- ![image](https://example.com/image.png)\n", "local file"},
		{"dangerous URL", "# Title\n- [bad](javascript:alert%281%29)\n", "unsupported link"},
		{"data URL", "# Title\n- [bad](data:text/html,hello)\n", "unsupported link"},
		{"relative URL", "# Title\n- [bad](../thing.md)\n", "unsupported link"},
		{"autolink", "# Title\n- <https://example.com>\n", "autolinks"},
		{"reference definition", "# Title\n- See [source][s].\n\n[s]: https://example.com\n", "reference-definition"},
		{"fence", "# Title\n- Point.\n\n  ```go\n  x := 1\n  ```\n", "only mermaid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseOutline([]byte(test.source))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("wanted %q, got %v", test.want, err)
			}
		})
	}
}

func TestWrappedTextAndSourceLines(t *testing.T) {
	out, err := parseOutline([]byte("\xef\xbb\xbf# Wrapped\r\n\r\n- A point\r\n  spans lines. {#point}\r\n  - A child.\r\n\r\n* A second root.\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Flat) != 3 || out.Flat[0].Line != 3 || out.Flat[0].Text != "A point spans lines." || out.Flat[2].Number != "2" {
		t.Fatalf("wrong source mapping: %+v", out.Flat)
	}
}

func TestRenderedContentEscapesMarkup(t *testing.T) {
	out, err := parseOutline([]byte("# A & B\n- **Bold**, *emphasis*, `</script><img src=x>` and [source](https://example.com/?a=1&amp;b=2 \"Source title\").\n"))
	if err != nil {
		t.Fatal(err)
	}
	var result bytes.Buffer
	if err := renderOutline(&result, out, 1, "."); err != nil {
		t.Fatal(err)
	}
	page := result.String()
	for _, want := range []string{"<strong>Bold</strong>", "<em>emphasis</em>", "&lt;/script&gt;&lt;img src=x&gt;", "<title>A &amp; B", `href="https://example.com/?a=1&amp;b=2"`, `title="Source title"`} {
		if !strings.Contains(page, want) {
			t.Errorf("missing escaped content %q", want)
		}
	}
	for _, forbidden := range []string{"<script src=", "<link rel=", "<img src=x>"} {
		if strings.Contains(page, forbidden) {
			t.Errorf("unexpected markup %q", forbidden)
		}
	}
}

func TestCLIFilesAndStdin(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "input.md"), filepath.Join(dir, "output.html")
	os.WriteFile(input, []byte("# Title\n- Root. {#root}\n  - Child.\n"), 0644)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"render", input, "-o", output, "--depth", "2"}, nil, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	page, _ := os.ReadFile(output)
	if !bytes.Contains(page, []byte("<details open>")) {
		t.Fatal("depth flag did not open the parent")
	}
	stdout.Reset()
	if err := run([]string{"refs", input}, nil, &stdout, &stderr); err != nil || !strings.Contains(stdout.String(), "[1]\t2\troot") {
		t.Fatalf("refs failed: %s %v", stdout.String(), err)
	}
	stdout.Reset()
	if err := run([]string{"render", "-", "-o", "-"}, strings.NewReader("# Stdin\n- Point.\n"), &stdout, &stderr); err != nil || !strings.HasPrefix(stdout.String(), "<!doctype html>") {
		t.Fatalf("stdin failed: %v", err)
	}
	if err := run([]string{"render", input, "-o", input}, nil, &stdout, &stderr); err == nil {
		t.Fatal("allowed overwriting source")
	}
	// A parse error must leave an existing rendered document untouched.
	os.WriteFile(input, []byte("# Invalid\n- [Missing](#missing).\n"), 0644)
	if err := run([]string{"render", input, "-o", output}, nil, &stdout, &stderr); err == nil {
		t.Fatal("allowed unresolved reference")
	}
	after, _ := os.ReadFile(output)
	if !bytes.Equal(page, after) {
		t.Fatal("failed render changed existing output")
	}
}

func TestFilesystemAliasCannotOverwriteSource(t *testing.T) {
	dir := t.TempDir()
	source, alias := filepath.Join(dir, "source.html"), filepath.Join(dir, "alias.html")
	content := []byte("# Source\n- Keep me.\n")
	os.WriteFile(source, content, 0644)
	if err := os.Symlink(source, alias); err != nil {
		t.Skip(err)
	}
	err := run([]string{"render", source, "-o", alias}, nil, ioDiscard{}, ioDiscard{})
	if err == nil || !strings.Contains(err.Error(), "different files") {
		t.Fatalf("alias accepted: %v", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func TestEveryExample(t *testing.T) {
	files, _ := filepath.Glob("examples/*.md")
	if len(files) < 7 {
		t.Fatal("missing examples")
	}
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		out, err := parseOutline(source)
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		if err := renderOutline(ioDiscard{}, out, 1, filepath.Dir(file)); err != nil {
			t.Errorf("%s: %v", file, err)
		}
	}
}
