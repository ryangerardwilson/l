package tree

import (
	"fmt"
	"io"
	"math"
)

type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

type RenderConfig struct {
	Color bool
}

func Render(out io.Writer, root *Node, stats Stats, cfg RenderConfig) error {
	if root == nil {
		return nil
	}
	return renderLine(out, root, "", false, true, stats, cfg)
}

func renderLine(out io.Writer, node *Node, prefix string, last bool, root bool, stats Stats, cfg RenderConfig) error {
	connector := ""
	childPrefix := ""
	if !root {
		if last {
			connector = "└── "
			childPrefix = prefix + "    "
		} else {
			connector = "├── "
			childPrefix = prefix + "│   "
		}
	}

	line := fmt.Sprintf("%s %s%s%s", node.Type, prefix, connector, node.Name)
	if detail := detailFor(node); detail != "" {
		line += " " + detail
	}
	line = colorize(line, node, stats, cfg)
	if _, err := fmt.Fprintln(out, line); err != nil {
		return err
	}

	for i, child := range node.Children {
		if err := renderLine(out, child, childPrefix, i == len(node.Children)-1, false, stats, cfg); err != nil {
			return err
		}
	}
	return nil
}

func detailFor(node *Node) string {
	switch node.Type {
	case TypeFile:
		return fmt.Sprintf("%s %dL", humanSize(node.Size), node.Lines)
	case TypeBinary:
		return humanSize(node.Size)
	default:
		return ""
	}
}

func humanSize(size int64) string {
	if size < 0 {
		return "?B"
	}
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	value := float64(size)
	unit := "KB"
	for _, next := range units {
		value = value / 1024
		unit = next
		if math.Abs(value) < 1024 {
			break
		}
	}
	return fmt.Sprintf("%.1f%s", value, unit)
}

func colorize(line string, node *Node, stats Stats, cfg RenderConfig) string {
	if !cfg.Color {
		return line
	}

	style := styleFor(node, stats)
	if style == "" {
		return line
	}
	return style + line + "\x1b[0m"
}

func styleFor(node *Node, stats Stats) string {
	switch node.Type {
	case TypeDirectory:
		return "\x1b[36m"
	case TypeBinary:
		if stats.MaxBinarySize > 0 && node.Size >= stats.MaxBinarySize*3/4 {
			return "\x1b[1;95m"
		}
		return "\x1b[35m"
	case TypeFile:
		max := stats.MaxTextLines
		if max <= 0 {
			return "\x1b[32m"
		}
		ratio := float64(node.Lines) / float64(max)
		if max >= 1000 && ratio >= 0.75 {
			return "\x1b[1;91m"
		}
		if max >= 500 && ratio >= 0.35 {
			return "\x1b[93m"
		}
		if max >= 1000 && ratio <= 0.05 {
			return "\x1b[2m"
		}
		return "\x1b[32m"
	default:
		return ""
	}
}
