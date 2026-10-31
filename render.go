package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"html"
	"html/template"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

//go:embed web/*
var assets embed.FS

type linkRenderer struct{ ids map[string]*Point }

func (r *linkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.render)
}

func (r *linkRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	link := node.(*ast.Link)
	dest := string(link.Destination)
	var target *Point
	if strings.HasPrefix(dest, "#") {
		id, _ := url.PathUnescape(dest[1:])
		target = r.ids[id]
	}
	if entering {
		if target != nil {
			fmt.Fprintf(w, `<a class="xref" href="#p-%s" title="Go to [%s]">`, target.Number, target.Number)
		} else {
			fmt.Fprintf(w, `<a href="%s" rel="noreferrer"`, html.EscapeString(string(util.URLEscape(link.Destination, true))))
			if len(link.Title) > 0 {
				fmt.Fprintf(w, ` title="%s"`, html.EscapeString(string(link.Title)))
			}
			w.WriteString(">")
		}
	} else {
		if target != nil {
			fmt.Fprintf(w, ` <span class="ref-number">[%s]</span>`, target.Number)
		}
		w.WriteString("</a>")
	}
	return ast.WalkContinue, nil
}

type mediaRenderer struct{ baseDir string }

func (r *mediaRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.image)
	reg.Register(ast.KindFencedCodeBlock, r.diagram)
}

func (r *mediaRenderer) image(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	image := node.(*ast.Image)
	path, err := url.Parse(string(util.URLEscape(image.Destination, true)))
	if err != nil {
		return ast.WalkStop, err
	}
	filename := filepath.FromSlash(path.Path)
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(r.baseDir, filename)
	}
	kind := strings.Split(mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))), ";")[0]
	switch kind {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/svg+xml", "image/avif":
	default:
		return ast.WalkStop, fmt.Errorf("image %q: use PNG, JPEG, GIF, WebP, SVG, or AVIF", filename)
	}
	f, err := os.Open(filename)
	if err != nil {
		return ast.WalkStop, fmt.Errorf("image: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 20*1024*1024+1))
	if err != nil {
		return ast.WalkStop, err
	}
	if len(data) > 20*1024*1024 {
		return ast.WalkStop, fmt.Errorf("image %q exceeds 20 MiB", filename)
	}
	fmt.Fprintf(w, `<span class="image-block"><button type="button" class="image-size" aria-pressed="false">Full size</button><span class="image-frame"><img src="data:%s;base64,%s" alt="%s" loading="lazy"></span></span>`, kind, base64.StdEncoding.EncodeToString(data), html.EscapeString(string(image.Text(source))))
	return ast.WalkSkipChildren, nil
}

func (r *mediaRenderer) diagram(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		code := node.(*ast.FencedCodeBlock)
		fmt.Fprintf(w, `<div class="diagram"><pre>%s</pre></div>`, html.EscapeString(string(code.Lines().Value(source))))
	}
	return ast.WalkSkipChildren, nil
}

func renderOutline(w io.Writer, outline *Outline, depth int, baseDir string) error {
	md := markdown()
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&linkRenderer{outline.ByID}, 100), util.Prioritized(&mediaRenderer{baseDir}, 100)))
	var tree strings.Builder
	hasDiagrams := false
	var renderPoints func([]*Point, int) error
	renderPoints = func(points []*Point, level int) error {
		tree.WriteString(`<ul class="points">`)
		for _, p := range points {
			var content bytes.Buffer
			if err := md.Convert([]byte(p.Text), &content); err != nil {
				return err
			}
			body := strings.TrimSpace(content.String())
			body = strings.TrimSuffix(strings.TrimPrefix(body, "<p>"), "</p>")
			fmt.Fprintf(&tree, `<li class="point" id="p-%s">`, p.Number)
			branch := len(p.Children) > 0 || len(p.Content) > 0
			if branch {
				open := ""
				if depth == 0 || level < depth {
					open = " open"
				}
				fmt.Fprintf(&tree, `<details%s><summary>`, open)
			} else {
				tree.WriteString(`<div class="leaf-row">`)
			}
			fmt.Fprintf(&tree, `<button class="copy-markdown" type="button" data-markdown="%s" title="Copy Markdown for [%s]" aria-label="Copy Markdown for point [%s]">md <svg viewBox="0 0 16 16" aria-hidden="true"><rect x="5" y="5" width="8" height="9" rx="1.5"/><path d="M10 5V3a1 1 0 0 0-1-1H3a1 1 0 0 0-1 1v7a1 1 0 0 0 1 1h2"/></svg></button>`, html.EscapeString(p.Markdown), p.Number, p.Number)
			fmt.Fprintf(&tree, `<button class="copy-point" type="button" data-copy="[%s]" aria-label="Copy point ID [%s]" title="Copy [%s]"><span class="point-number">[%s]</span><svg viewBox="0 0 16 16" aria-hidden="true"><rect x="5" y="5" width="8" height="9" rx="1.5"/><path d="M10 5V3a1 1 0 0 0-1-1H3a1 1 0 0 0-1 1v7a1 1 0 0 0 1 1h2"/></svg></button>`, p.Number, p.Number, p.Number, p.Number)
			tree.WriteString(body)
			if branch {
				tree.WriteString(`</summary>`)
				if len(p.Content) > 0 {
					tree.WriteString(`<div class="point-content">`)
					for _, visual := range p.Content {
						var rendered bytes.Buffer
						if err := md.Convert([]byte(visual.Markdown), &rendered); err != nil {
							return fmt.Errorf("line %d [%s]: %w", p.Line, p.Number, err)
						}
						tree.WriteString(rendered.String())
						hasDiagrams = hasDiagrams || visual.Kind == "mermaid"
					}
					tree.WriteString(`</div>`)
				}
				if len(p.Children) > 0 {
					if err := renderPoints(p.Children, level+1); err != nil {
						return err
					}
				}
				tree.WriteString(`</details>`)
			} else {
				tree.WriteString(`</div>`)
			}
			tree.WriteString(`</li>`)
		}
		tree.WriteString(`</ul>`)
		return nil
	}
	if err := renderPoints(outline.Points, 1); err != nil {
		return err
	}
	page, err := assets.ReadFile("web/page.html")
	if err != nil {
		return err
	}
	css, _ := assets.ReadFile("web/style.css")
	js, _ := assets.ReadFile("web/reader.js")
	theme, _ := assets.ReadFile("web/theme.js")
	t, err := template.New("page").Parse(string(page))
	if err != nil {
		return err
	}
	return t.Execute(w, struct {
		Title    string
		Tree     template.HTML
		CSS      template.CSS
		JS       template.JS
		Diagrams bool
		Theme    template.JS
		Export   struct{ Filename, Source string }
	}{outline.Title, template.HTML(tree.String()), template.CSS(css), template.JS(js), hasDiagrams, template.JS(theme), struct{ Filename, Source string }{outline.Filename, base64.StdEncoding.EncodeToString(outline.Source)}})
}
