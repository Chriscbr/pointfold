package main

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// A point has its own Markdown, optional visual blocks, and ordinary child points.
type Point struct {
	Text     string
	Markdown string
	Content  []Visual
	StableID string
	Number   string
	Line     int
	Children []*Point
}

type Visual struct {
	Kind     string
	Markdown string
}

func markdown() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.Table))
}

type Outline struct {
	Title    string
	Points   []*Point
	ByID     map[string]*Point
	Flat     []*Point
	MaxDepth int
}

var idSuffix = regexp.MustCompile(`(?:^|\s+)\{#([a-z][a-z0-9-]*)\}\s*$`)
var idLike = regexp.MustCompile(`\{#[^{}]*\}\s*$`)

func parseOutline(source []byte) (*Outline, error) {
	if !utf8.Valid(source) {
		return nil, fmt.Errorf("input must be UTF-8")
	}
	source = bytes.TrimPrefix(source, []byte("\xef\xbb\xbf"))
	source = bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n"))
	context := parser.NewContext()
	root := markdown().Parser().Parse(text.NewReader(source), parser.WithContext(context))
	if len(context.References()) > 0 {
		return nil, fmt.Errorf("use explicit inline links; reference-definition blocks are unsupported")
	}
	out := &Outline{ByID: make(map[string]*Point)}
	titleSeen := false
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		switch v := n.(type) {
		case *ast.Heading:
			if titleSeen || len(out.Points) > 0 || v.Level != 1 {
				return nil, at(source, n, "use one H1 title before the outline; put sections in bullets")
			}
			if err := validateInline(n, source, nil, false); err != nil {
				return nil, err
			}
			// The title is plain text in the viewer; code and emphasis lose only their styling.
			out.Title = string(n.Text(source))
			titleSeen = true
		case *ast.List:
			points, err := parseList(v, source, out, "", 1, len(out.Points))
			if err != nil {
				return nil, err
			}
			out.Points = append(out.Points, points...)
		default:
			return nil, at(source, n, "expected a nested bullet list, not %s", n.Kind())
		}
	}
	if !titleSeen || strings.TrimSpace(out.Title) == "" {
		return nil, fmt.Errorf("add a title on the first line, for example: # My topic")
	}
	if len(out.Points) == 0 {
		return nil, fmt.Errorf("add at least one bullet point below the title")
	}
	// Resolve only after collecting every ID, so forward references are allowed.
	for _, p := range out.Flat {
		md := markdown().Parser().Parse(text.NewReader([]byte(p.Text)))
		if err := validateInline(md, []byte(p.Text), out.ByID, false); err != nil {
			return nil, fmt.Errorf("line %d [%s]: %w", p.Line, p.Number, err)
		}
		for _, visual := range p.Content {
			md := markdown().Parser().Parse(text.NewReader([]byte(visual.Markdown)))
			if err := validateInline(md, []byte(visual.Markdown), out.ByID, true); err != nil {
				return nil, fmt.Errorf("line %d [%s]: %w", p.Line, p.Number, err)
			}
		}
	}
	return out, nil
}

func parseList(list *ast.List, source []byte, out *Outline, prefix string, depth, offset int) ([]*Point, error) {
	if list.IsOrdered() {
		return nil, at(source, list, "use unordered bullets; outline numbers are generated")
	}
	if depth > 64 {
		return nil, at(source, list, "nesting exceeds the prototype limit of 64 levels")
	}
	if depth > out.MaxDepth {
		out.MaxDepth = depth
	}
	var points []*Point
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		block := item.FirstChild()
		if block == nil || (block.Kind() != ast.KindParagraph && block.Kind() != ast.KindTextBlock) {
			return nil, at(source, item, "a point must start with its own text")
		}
		raw := blockText(block, source)
		p := &Point{Number: prefix + strconv.Itoa(offset+len(points)+1), Line: lineOf(source, block)}
		if match := idSuffix.FindStringSubmatchIndex(raw); match != nil {
			// Only a literal trailing text suffix is metadata, never inline code or a link.
			last, ok := block.LastChild().(*ast.Text)
			if !ok || !idSuffix.Match(last.Segment.Value(source)) {
				return nil, at(source, block, "place {#stable-id} after the point text, outside Markdown formatting")
			}
			p.StableID = raw[match[2]:match[3]]
			raw = raw[:match[0]]
			if previous := out.ByID[p.StableID]; previous != nil {
				return nil, at(source, block, "duplicate ID %q (first used on line %d)", p.StableID, previous.Line)
			}
			out.ByID[p.StableID] = p
		} else if idLike.MatchString(raw) {
			return nil, at(source, block, "invalid stable ID; use {#lowercase-words-and-digits}")
		}
		p.Text = strings.TrimSpace(raw)
		if p.Text == "" {
			return nil, at(source, block, "a point needs text before its ID")
		}
		p.Markdown = strings.TrimSpace(blockMarkdown(block, source))
		if p.StableID != "" {
			p.Markdown = strings.TrimSpace(idSuffix.ReplaceAllString(p.Markdown, ""))
		}
		// An image-only bullet uses its alt text as the collapsed label.
		inline := markdown().Parser().Parse(text.NewReader([]byte(p.Text))).FirstChild()
		if image := onlyImage(inline); image != nil {
			label := strings.TrimSpace(string(image.Text([]byte(p.Text))))
			if label == "" {
				return nil, at(source, block, "an image-only point needs descriptive alt text")
			}
			p.Content = append(p.Content, Visual{"image", p.Markdown})
			p.Text = escapeMarkdown(label)
		}
		out.Flat = append(out.Flat, p)
		for child := block.NextSibling(); child != nil; child = child.NextSibling() {
			if list, ok := child.(*ast.List); ok {
				children, err := parseList(list, source, out, p.Number+".", depth+1, len(p.Children))
				if err != nil {
					return nil, err
				}
				p.Children = append(p.Children, children...)
				continue
			}
			visual, err := parseVisual(child, source)
			if err != nil {
				return nil, err
			}
			p.Content = append(p.Content, visual)
			p.Markdown += "\n\n" + visual.Markdown
		}
		points = append(points, p)
	}
	return points, nil
}

func onlyImage(block ast.Node) *ast.Image {
	if block == nil || (block.Kind() != ast.KindParagraph && block.Kind() != ast.KindTextBlock) {
		return nil
	}
	image, ok := block.FirstChild().(*ast.Image)
	if ok && image.NextSibling() == nil {
		return image
	}
	return nil
}

func escapeMarkdown(value string) string {
	var out strings.Builder
	for _, char := range value {
		if strings.ContainsRune("\\`*_{}[]()#+-.!|<>~", char) {
			out.WriteByte('\\')
		}
		out.WriteRune(char)
	}
	return out.String()
}

func parseVisual(node ast.Node, source []byte) (Visual, error) {
	switch v := node.(type) {
	case *extast.Table:
		start := v.FirstChild().Pos()
		end := v.LastChild().Pos()
		if newline := bytes.IndexByte(source[end:], '\n'); newline >= 0 {
			end += newline + 1
		} else {
			end = len(source)
		}
		if v.ChildCount() == 1 { // A header-only table still includes its delimiter row.
			if newline := bytes.IndexByte(source[end:], '\n'); newline >= 0 {
				end += newline + 1
			} else {
				end = len(source)
			}
		}
		lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
		indent := string(source[lineStart:start])
		lines := strings.Split(strings.TrimSuffix(string(source[start:end]), "\n"), "\n")
		for i := 1; i < len(lines); i++ {
			lines[i] = strings.TrimPrefix(lines[i], indent)
		}
		return Visual{"table", strings.Join(lines, "\n")}, nil
	case *ast.FencedCodeBlock:
		language := string(v.Language(source))
		if language != "mermaid" && language != "pele" {
			return Visual{}, at(source, node, "only mermaid or pele code blocks are supported as visual blocks")
		}
		code := string(v.Lines().Value(source))
		fence := "```"
		for strings.Contains(code, fence) {
			fence += "`"
		}
		return Visual{language, fence + language + "\n" + strings.TrimSuffix(code, "\n") + "\n" + fence}, nil
	default:
		if onlyImage(node) != nil {
			return Visual{"image", strings.TrimSpace(blockMarkdown(node, source))}, nil
		}
		return Visual{}, at(source, node, "a point has one text block; use child points for more prose, or attach a table, image, mermaid, or pele block")
	}
}

func blockMarkdown(n ast.Node, source []byte) string {
	var parts []string
	for i := 0; i < n.Lines().Len(); i++ {
		segment := n.Lines().At(i)
		parts = append(parts, strings.TrimSuffix(string(segment.Value(source)), "\n"))
	}
	return strings.Join(parts, "\n")
}

func blockText(n ast.Node, source []byte) string {
	var parts []string
	for i := 0; i < n.Lines().Len(); i++ {
		segment := n.Lines().At(i)
		parts = append(parts, strings.TrimSpace(string(segment.Value(source))))
	}
	return strings.Join(parts, " ")
}

func lineOf(source []byte, n ast.Node) int {
	if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
		return bytes.Count(source[:n.Lines().At(0).Start], []byte("\n")) + 1
	}
	if child := n.FirstChild(); child != nil {
		return lineOf(source, child)
	}
	return 1
}

func at(source []byte, n ast.Node, format string, args ...any) error {
	return fmt.Errorf("line %d: %s", lineOf(source, n), fmt.Sprintf(format, args...))
}

func validateInline(root ast.Node, source []byte, ids map[string]*Point, images bool) error {
	return ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.RawHTML, *ast.HTMLBlock, *ast.AutoLink:
			return ast.WalkStop, fmt.Errorf("raw HTML and autolinks are unsupported; use explicit Markdown links")
		case *ast.Image:
			if !images {
				return ast.WalkStop, fmt.Errorf("place images in their own paragraph under a point, or use an image-only bullet")
			}
			dest := string(v.Destination)
			u, err := url.Parse(dest)
			if err != nil || dest == "" || u.Scheme != "" || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
				return ast.WalkStop, fmt.Errorf("image %q must be a local file path; local images are embedded when rendering", dest)
			}
		case *ast.Link:
			dest := string(v.Destination)
			if strings.HasPrefix(dest, "#") {
				id, err := url.PathUnescape(strings.TrimPrefix(dest, "#"))
				if err != nil || id == "" {
					return ast.WalkStop, fmt.Errorf("invalid cross-reference %q", dest)
				}
				if ids != nil && ids[id] == nil {
					return ast.WalkStop, fmt.Errorf("unresolved cross-reference %q; add {#%s} to its target", dest, id)
				}
			} else {
				u, err := url.Parse(dest)
				if err != nil || (u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "mailto") || ((u.Scheme == "https" || u.Scheme == "http") && u.Host == "") {
					return ast.WalkStop, fmt.Errorf("unsupported link %q; use https://, http://, mailto:, or #stable-id", dest)
				}
			}
		}
		return ast.WalkContinue, nil
	})
}
