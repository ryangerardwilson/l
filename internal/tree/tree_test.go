package tree

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildUsesCommonSenseIgnoresAndHiddenRules(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "src"))
	mustMkdir(t, filepath.Join(root, "node_modules"))
	mustMkdir(t, filepath.Join(root, "__pycache__"))
	mustMkdir(t, filepath.Join(root, ".git"))
	mustWrite(t, filepath.Join(root, "src", "app.go"), "one\ntwo\n")
	mustWrite(t, filepath.Join(root, "node_modules", "dep.js"), "ignored\n")
	mustWrite(t, filepath.Join(root, "__pycache__", "x.pyc"), "\x00binary")
	mustWrite(t, filepath.Join(root, ".env"), "hidden\n")

	node, _, err := Build(Config{
		Root:              root,
		MaxDepth:          2,
		UseDefaultIgnores: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := renderNoColor(t, node)
	if strings.Contains(output, "node_modules") || strings.Contains(output, "__pycache__") || strings.Contains(output, ".git") {
		t.Fatalf("expected generated/dependency dirs to be ignored:\n%s", output)
	}
	if strings.Contains(output, ".env") {
		t.Fatalf("expected hidden file to be hidden by default:\n%s", output)
	}
	if !strings.Contains(output, "src") || !strings.Contains(output, "app.go") {
		t.Fatalf("expected source files to be shown:\n%s", output)
	}
}

func TestBuildCanIncludeHiddenAndDisableIgnores(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "node_modules"))
	mustWrite(t, filepath.Join(root, "node_modules", "dep.js"), "dep\n")
	mustWrite(t, filepath.Join(root, ".env"), "hidden\n")

	node, _, err := Build(Config{
		Root:              root,
		MaxDepth:          2,
		ShowHidden:        true,
		UseDefaultIgnores: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := renderNoColor(t, node)
	for _, want := range []string{"node_modules", "dep.js", ".env"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output:\n%s", want, output)
		}
	}
}

func TestBuildClassifiesTextAndBinaryAndCountsFinalLine(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "text.txt"), "one\ntwo")
	mustWrite(t, filepath.Join(root, "empty.txt"), "")
	mustWrite(t, filepath.Join(root, "blob.bin"), "\x00abc")

	node, _, err := Build(Config{
		Root:              root,
		MaxDepth:          1,
		UseDefaultIgnores: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := renderNoColor(t, node)
	for _, want := range []string{
		"F ├── empty.txt 0B 0L",
		"F └── text.txt 7B 2L",
		"B ├── blob.bin 4B",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output:\n%s", want, output)
		}
	}
}

func TestDepthLimitsChildren(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "src", "nested"))
	mustWrite(t, filepath.Join(root, "src", "nested", "deep.go"), "deep\n")

	node, _, err := Build(Config{
		Root:              root,
		MaxDepth:          1,
		UseDefaultIgnores: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := renderNoColor(t, node)
	if !strings.Contains(output, "src") {
		t.Fatalf("expected depth 1 child:\n%s", output)
	}
	if strings.Contains(output, "nested") || strings.Contains(output, "deep.go") {
		t.Fatalf("expected depth limit to hide grandchildren:\n%s", output)
	}
}

func TestColorHighlightsLargeTextFiles(t *testing.T) {
	huge := &Node{Name: "huge.go", Type: TypeFile, Size: 10, Lines: 20000}
	line := colorize("F huge.go 10B 20000L", huge, Stats{MaxTextLines: 20000}, RenderConfig{Color: true})
	if !strings.Contains(line, "\x1b[1;91m") {
		t.Fatalf("expected huge file to be bold bright red, got %q", line)
	}

	small := &Node{Name: "small.go", Type: TypeFile, Size: 10, Lines: 1000}
	line = colorize("F small.go 10B 1000L", small, Stats{MaxTextLines: 20000}, RenderConfig{Color: true})
	if !strings.Contains(line, "\x1b[2m") {
		t.Fatalf("expected small relative file to be dim, got %q", line)
	}
}

func renderNoColor(t *testing.T, node *Node) string {
	t.Helper()
	var stats Stats
	collectStats(node, &stats)
	var buf bytes.Buffer
	if err := Render(&buf, node, stats, RenderConfig{}); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
