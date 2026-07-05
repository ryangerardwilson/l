# l

`l` is a compact tree view for spotting refactoring damage in a codebase.

It hides dependency and cache noise by default, colors entries as directories,
text files, or binaries, shows file size and line count, and highlights large
visible files relative to the rest of the tree. Tree glyphs render gray.
Directories render white, text files render yellow, and binaries render cyan.
Text-file brightness scales by visible line counts; binary brightness uses the
same relative logic on file size. Files under `200L` and binaries under
`100KB` share one low-intensity color so small entries do not create noisy
contrast.

## Install

With Go:

```bash
go install github.com/ryangerardwilson/l/cmd/l@latest
```

From a checkout:

```bash
go build -o ./bin/l ./cmd/l
```

## Usage

```bash
l
l ~/Apps/my-app -d 5
l -a ~/Apps/my-app
l -f ~/Apps/my-app
l . --depth=*
l . --no-ignore
l . --color=always
l . --no-animate
```

Output:

```text
.
├── internal
│   ├── app.go 4.2KB 130L
│   └── generated_dump.go 680.0KB 20000L
└── model.bin 42.0MB
```

The default depth is `2`. Use `-d *` for unlimited depth. Use `-f` to show
text files only; it hides binaries and prunes directories that do not contain
visible text files.

## Defaults

`l` ignores common generated or low-signal directories by default, including:

- `node_modules`
- `__pycache__`
- `.git`
- `bin`
- `dist`
- `build`
- `target`
- `coverage`
- `venv` and `.venv`
- `.next`, `.turbo`, and common cache directories

Use `--no-ignore` to show them anyway. Use `--ignore <name-or-glob>` to add
more ignores for a run.

Color is `auto` by default, which means colors are emitted only when stdout is
a terminal. Use `--color=always` or `--no-color` to override that.

Animation is also `auto` by default. On an interactive terminal, `l` renders
each line as a quick branch-then-label cascade. Redirected or piped output stays
plain and deterministic. Use `--no-animate` to disable it, or
`--animate=always` / `--animate=never` for explicit control.

## Development

```bash
go test ./...
go run ./cmd/l --help
go run ./cmd/l . -d 2 --color=always --animate=always
```
