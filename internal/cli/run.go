package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ryangerardwilson/l/internal/tree"
	"github.com/ryangerardwilson/l/internal/version"
)

type options struct {
	config        tree.Config
	colorMode     tree.ColorMode
	animationMode tree.AnimationMode
	help          bool
	version       bool
}

func Main(args []string, stdout io.Writer, stderr io.Writer) int {
	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "l: %s\n\n", err)
		writeHelp(stderr)
		return 2
	}
	if opts.help {
		writeHelp(stdout)
		return 0
	}
	if opts.version {
		fmt.Fprintf(stdout, "l %s\n", version.Version)
		return 0
	}

	root, stats, err := tree.Build(opts.config)
	if err != nil {
		fmt.Fprintf(stderr, "l: %s\n", err)
		return 1
	}

	if err := tree.Render(stdout, root, stats, tree.RenderConfig{
		Color:          shouldColor(stdout, opts.colorMode),
		Animate:        shouldAnimate(stdout, opts.animationMode),
		AnimationDelay: tree.DefaultAnimationDelay,
	}); err != nil {
		fmt.Fprintf(stderr, "l: %s\n", err)
		return 1
	}
	return 0
}

func parseArgs(args []string) (options, error) {
	opts := options{
		config: tree.Config{
			Root:              ".",
			MaxDepth:          2,
			UseDefaultIgnores: true,
		},
		colorMode:     tree.ColorAuto,
		animationMode: tree.AnimationAuto,
	}
	var paths []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "help" || arg == "-h" || arg == "--help":
			opts.help = true
		case arg == "version" || arg == "--version":
			opts.version = true
		case arg == "-a" || arg == "--all":
			opts.config.ShowHidden = true
		case arg == "-f" || arg == "--files":
			opts.config.FilesOnly = true
		case arg == "-d" || arg == "--depth" || arg == "--level":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%s requires a depth", arg)
			}
			depth, err := parseDepth(args[i+1])
			if err != nil {
				return opts, err
			}
			opts.config.MaxDepth = depth
			i++
		case strings.HasPrefix(arg, "--depth=") || strings.HasPrefix(arg, "--level="):
			value := strings.TrimPrefix(strings.TrimPrefix(arg, "--depth="), "--level=")
			depth, err := parseDepth(value)
			if err != nil {
				return opts, err
			}
			opts.config.MaxDepth = depth
		case arg == "--color":
			if i+1 >= len(args) {
				return opts, errors.New("--color requires auto, always, or never")
			}
			mode, err := parseColorMode(args[i+1])
			if err != nil {
				return opts, err
			}
			opts.colorMode = mode
			i++
		case strings.HasPrefix(arg, "--color="):
			mode, err := parseColorMode(strings.TrimPrefix(arg, "--color="))
			if err != nil {
				return opts, err
			}
			opts.colorMode = mode
		case arg == "--no-color":
			opts.colorMode = tree.ColorNever
		case arg == "--animate":
			if i+1 >= len(args) {
				return opts, errors.New("--animate requires auto, always, or never")
			}
			mode, err := parseAnimationMode(args[i+1])
			if err != nil {
				return opts, err
			}
			opts.animationMode = mode
			i++
		case strings.HasPrefix(arg, "--animate="):
			mode, err := parseAnimationMode(strings.TrimPrefix(arg, "--animate="))
			if err != nil {
				return opts, err
			}
			opts.animationMode = mode
		case arg == "--no-animate":
			opts.animationMode = tree.AnimationNever
		case arg == "--no-ignore":
			opts.config.UseDefaultIgnores = false
		case arg == "--ignore":
			if i+1 >= len(args) {
				return opts, errors.New("--ignore requires a name or glob")
			}
			opts.config.ExtraIgnores = appendIgnores(opts.config.ExtraIgnores, args[i+1])
			i++
		case strings.HasPrefix(arg, "--ignore="):
			opts.config.ExtraIgnores = appendIgnores(opts.config.ExtraIgnores, strings.TrimPrefix(arg, "--ignore="))
		case arg == "--":
			paths = append(paths, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Errorf("unknown flag: %s", arg)
		default:
			paths = append(paths, arg)
		}
	}

	if len(paths) > 1 {
		return opts, errors.New("accepts at most one path")
	}
	if len(paths) == 1 {
		opts.config.Root = paths[0]
	}
	return opts, nil
}

func parseDepth(value string) (int, error) {
	if value == "*" {
		return -1, nil
	}
	depth, err := strconv.Atoi(value)
	if err != nil || depth < 0 {
		return 0, errors.New("depth must be a non-negative integer or *")
	}
	return depth, nil
}

func parseColorMode(value string) (tree.ColorMode, error) {
	switch tree.ColorMode(value) {
	case tree.ColorAuto, tree.ColorAlways, tree.ColorNever:
		return tree.ColorMode(value), nil
	default:
		return tree.ColorAuto, errors.New("--color must be auto, always, or never")
	}
}

func parseAnimationMode(value string) (tree.AnimationMode, error) {
	switch tree.AnimationMode(value) {
	case tree.AnimationAuto, tree.AnimationAlways, tree.AnimationNever:
		return tree.AnimationMode(value), nil
	default:
		return tree.AnimationAuto, errors.New("--animate must be auto, always, or never")
	}
}

func appendIgnores(existing []string, value string) []string {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			existing = append(existing, part)
		}
	}
	return existing
}

func shouldColor(out io.Writer, mode tree.ColorMode) bool {
	switch mode {
	case tree.ColorAlways:
		return true
	case tree.ColorNever:
		return false
	default:
		return isTerminal(out)
	}
}

func shouldAnimate(out io.Writer, mode tree.AnimationMode) bool {
	switch mode {
	case tree.AnimationAlways:
		return true
	case tree.AnimationNever:
		return false
	default:
		return isTerminal(out) && os.Getenv("TERM") != "dumb"
	}
}

func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func writeHelp(out io.Writer) {
	fmt.Fprint(out, `l - compact tree view for seeing refactoring damage

Usage:
  l [path] [flags]

Flags:
  -a, --all              include hidden files and directories
  -f, --files            show text files only; hide binaries and empty dirs
  -d, --depth <n|*>      max tree depth from root, or * for unlimited (default 2)
      --ignore <pattern> add a basename or glob ignore; repeat or comma-separate
      --no-ignore        disable default dependency/cache/build ignores
      --color <mode>     auto, always, or never (default auto)
      --no-color         disable color
      --animate <mode>   auto, always, or never (default auto)
      --no-animate       disable tree animation
  -h, --help             show this help
      --version          show version

Default ignored directories include node_modules, __pycache__, .git, bin,
dist, build, target, coverage, venv, .venv, .next, .turbo, and common cache dirs.
`)
}
