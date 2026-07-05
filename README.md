# l

`l` is a compact tree view for spotting refactoring damage in a codebase.

It hides dependency and cache noise by default, marks entries as directories,
text files, or binaries, shows file size and line count, and color-highlights
large visible files relative to the rest of the tree.

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
l . --depth=*
l . --no-ignore
l . --color=always
```

Output:

```text
D .
D ├── internal
F │   ├── app.go 4.2KB 130L
F │   └── generated_dump.go 680.0KB 20000L
B └── model.bin 42.0MB
```

The default depth is `2`. Use `-d *` for unlimited depth.

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

## Development

```bash
go test ./...
go run ./cmd/l --help
go run ./cmd/l . -d 2 --color=always
```
