package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const version = "0.5.0"

const usage = `Pointfold — read ideas a point at a time.

  pointfold render INPUT.md [-o OUTPUT.html] [--depth N]
  pointfold check INPUT.md
  pointfold refs INPUT.md

render creates a local HTML file (default: beside INPUT.md).
--depth 1 shows only root points; 2 shows one child level; 0 opens all.
Use INPUT - for stdin; rendering stdin requires -o (or -o - for stdout).
refs maps current outline numbers to source lines and stable IDs.
Flags may appear before or after the input path.
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "pointfold:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Fprint(stdout, usage)
		return nil
	}
	if args[0] == "--version" {
		fmt.Fprintln(stdout, "pointfold "+version)
		return nil
	}
	command := args[0]
	if command != "render" && command != "check" && command != "refs" {
		return fmt.Errorf("unknown command %q; use pointfold --help", command)
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	output, depth := "", 1
	if command == "render" {
		fs.StringVar(&output, "o", "", "output HTML path, or - for stdout")
		fs.IntVar(&depth, "depth", 1, "visible levels; 0 opens all")
	}
	ordered, err := flagsFirst(args[1:], command)
	if err != nil {
		return err
	}
	if err := fs.Parse(ordered); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%s needs exactly one input file; use pointfold --help", command)
	}
	if depth < 0 {
		return fmt.Errorf("--depth must be zero or greater")
	}
	input := fs.Arg(0)
	var source []byte
	if input == "-" {
		source, err = io.ReadAll(stdin)
	} else {
		source, err = os.ReadFile(input)
	}
	if err != nil {
		return err
	}
	outline, err := parseOutline(source)
	if err != nil {
		return fmt.Errorf("%s: %w", input, err)
	}
	if input != "-" {
		outline.Filename = filepath.Base(input)
	}
	switch command {
	case "check":
		fmt.Fprintf(stdout, "OK: %s — %d points, %d levels, %d stable IDs\n", input, len(outline.Flat), outline.MaxDepth, len(outline.ByID))
		return nil
	case "refs":
		fmt.Fprintln(stdout, "POINT\tLINE\tSTABLE ID\tTEXT")
		for _, p := range outline.Flat {
			fmt.Fprintf(stdout, "[%s]\t%d\t%s\t%s\n", p.Number, p.Line, p.StableID, p.Text)
		}
		return nil
	}
	if output == "" {
		if input == "-" {
			return fmt.Errorf("rendering stdin requires -o OUTPUT.html (or -o -)")
		}
		output = strings.TrimSuffix(input, filepath.Ext(input)) + ".html"
	}
	// A bad render or a mistaken path must never truncate the Markdown source.
	if output != "-" {
		if strings.ToLower(filepath.Ext(output)) != ".html" && strings.ToLower(filepath.Ext(output)) != ".htm" {
			return fmt.Errorf("output must end in .html or .htm")
		}
		if input != "-" {
			inAbs, _ := filepath.Abs(input)
			outAbs, _ := filepath.Abs(output)
			same := inAbs == outAbs
			if a, e1 := os.Stat(input); e1 == nil {
				if b, e2 := os.Stat(output); e2 == nil {
					same = same || os.SameFile(a, b)
				}
			}
			if same {
				return fmt.Errorf("input and output must be different files")
			}
		}
	}
	var result bytes.Buffer
	baseDir := "."
	if input != "-" {
		baseDir = filepath.Dir(input)
	}
	if err := renderOutline(&result, outline, depth, baseDir); err != nil {
		return err
	}
	if output == "-" {
		_, err = stdout.Write(result.Bytes())
		return err
	}
	if err := atomicWrite(output, result.Bytes()); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "Rendered %d points → %s\n", len(outline.Flat), output)
	return nil
}

func flagsFirst(args []string, command string) ([]string, error) {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if command == "render" && (name == "o" || name == "depth") {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s requires a value", arg)
			}
			i++
			flags = append(flags, args[i])
		}
	}
	return append(append(flags, "--"), positional...), nil
}

func atomicWrite(path string, content []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".pointfold-*.html")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(content); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
