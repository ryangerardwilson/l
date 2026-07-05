package tree

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Type string

const (
	TypeDirectory Type = "D"
	TypeFile      Type = "F"
	TypeBinary    Type = "B"
)

type Config struct {
	Root              string
	MaxDepth          int
	ShowHidden        bool
	FilesOnly         bool
	UseDefaultIgnores bool
	ExtraIgnores      []string
}

type Node struct {
	Name     string
	Path     string
	Type     Type
	Size     int64
	Lines    int
	Children []*Node
}

type Stats struct {
	TextFileCount   int
	MinTextLines    int
	MaxTextLines    int
	BinaryFileCount int
	MinBinarySize   int64
	MaxBinarySize   int64
}

var defaultIgnoredDirs = map[string]struct{}{
	".cache":        {},
	".git":          {},
	".hg":           {},
	".mypy_cache":   {},
	".next":         {},
	".nuxt":         {},
	".parcel-cache": {},
	".pytest_cache": {},
	".ruff_cache":   {},
	".svn":          {},
	".turbo":        {},
	".venv":         {},
	".vite":         {},
	".vscode":       {},
	"__pycache__":   {},
	"bin":           {},
	"build":         {},
	"coverage":      {},
	"dist":          {},
	"env":           {},
	"node_modules":  {},
	"target":        {},
	"venv":          {},
}

var defaultIgnoredFiles = map[string]struct{}{
	".DS_Store": {},
}

func Build(cfg Config) (*Node, Stats, error) {
	root := cfg.Root
	if root == "" {
		root = "."
	}
	if cfg.MaxDepth < -1 {
		return nil, Stats{}, errors.New("depth must be -1 or greater")
	}

	info, err := os.Lstat(root)
	if err != nil {
		return nil, Stats{}, err
	}

	node, err := nodeForPath(root, displayName(root), info)
	if err != nil {
		return nil, Stats{}, err
	}

	if node.Type == TypeDirectory {
		if err := walkChildren(node, 0, cfg); err != nil {
			return nil, Stats{}, err
		}
	}
	if cfg.FilesOnly && !pruneFilesOnly(node) {
		return nil, Stats{}, nil
	}

	var stats Stats
	collectStats(node, &stats)
	return node, stats, nil
}

func displayName(path string) string {
	if path == "" {
		return "."
	}
	cleaned := filepath.Clean(path)
	if cleaned == "" {
		return "."
	}
	return cleaned
}

func nodeForPath(path string, name string, info os.FileInfo) (*Node, error) {
	node := &Node{
		Name: name,
		Path: path,
		Size: info.Size(),
	}
	if info.IsDir() {
		node.Type = TypeDirectory
		return node, nil
	}

	lines, binary, err := analyzeFile(path)
	if err != nil {
		node.Type = TypeBinary
		return node, nil
	}
	if binary {
		node.Type = TypeBinary
		return node, nil
	}

	node.Type = TypeFile
	node.Lines = lines
	return node, nil
}

func walkChildren(parent *Node, depth int, cfg Config) error {
	if cfg.MaxDepth >= 0 && depth >= cfg.MaxDepth {
		return nil
	}

	entries, err := os.ReadDir(parent.Path)
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]
		if left.IsDir() != right.IsDir() {
			return left.IsDir()
		}
		return strings.ToLower(left.Name()) < strings.ToLower(right.Name())
	})

	for _, entry := range entries {
		name := entry.Name()
		if shouldSkip(name, entry.IsDir(), cfg) {
			continue
		}

		path := filepath.Join(parent.Path, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		child, err := nodeForPath(path, name, info)
		if err != nil {
			continue
		}
		parent.Children = append(parent.Children, child)
		if child.Type == TypeDirectory {
			if err := walkChildren(child, depth+1, cfg); err != nil {
				return err
			}
		}
	}

	return nil
}

func pruneFilesOnly(node *Node) bool {
	switch node.Type {
	case TypeFile:
		return true
	case TypeBinary:
		return false
	case TypeDirectory:
		kept := node.Children[:0]
		for _, child := range node.Children {
			if pruneFilesOnly(child) {
				kept = append(kept, child)
			}
		}
		node.Children = kept
		return len(node.Children) > 0
	default:
		return false
	}
}

func shouldSkip(name string, isDir bool, cfg Config) bool {
	if !cfg.ShowHidden && strings.HasPrefix(name, ".") {
		return true
	}
	if cfg.UseDefaultIgnores {
		if isDir {
			if _, ok := defaultIgnoredDirs[name]; ok {
				return true
			}
		} else if _, ok := defaultIgnoredFiles[name]; ok {
			return true
		}
	}
	for _, pattern := range cfg.ExtraIgnores {
		if matchesIgnore(pattern, name) {
			return true
		}
	}
	return false
}

func matchesIgnore(pattern string, name string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if !strings.ContainsAny(pattern, "*?[") {
		return pattern == name
	}
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

func analyzeFile(path string) (int, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, true, err
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	lines := 0
	seenBytes := false
	lastByte := byte('\n')

	for {
		n, readErr := file.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			seenBytes = true
			if bytes.IndexByte(chunk, 0) >= 0 {
				return 0, true, nil
			}
			lines += bytes.Count(chunk, []byte{'\n'})
			lastByte = chunk[n-1]
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		return 0, true, readErr
	}

	if seenBytes && lastByte != '\n' {
		lines++
	}
	return lines, false, nil
}

func collectStats(node *Node, stats *Stats) {
	switch node.Type {
	case TypeFile:
		stats.TextFileCount++
		if stats.TextFileCount == 1 || node.Lines < stats.MinTextLines {
			stats.MinTextLines = node.Lines
		}
		if node.Lines > stats.MaxTextLines {
			stats.MaxTextLines = node.Lines
		}
	case TypeBinary:
		stats.BinaryFileCount++
		if stats.BinaryFileCount == 1 || node.Size < stats.MinBinarySize {
			stats.MinBinarySize = node.Size
		}
		if node.Size > stats.MaxBinarySize {
			stats.MaxBinarySize = node.Size
		}
	}
	for _, child := range node.Children {
		collectStats(child, stats)
	}
}
