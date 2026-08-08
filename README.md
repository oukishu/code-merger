# code-merger

A zero-dependency Go CLI tool that merges project source code into a single text file, designed for feeding project context to AI (code review, context injection, etc.).

## Features

- Pure standard library implementation, no third-party dependencies
- Automatically generates a project directory tree (with exclusion markers)
- Supports mixed processing of multiple directories and files
- Directory/file-level include/exclude rules (comma-separated, repeatable flags)
- Keyword filtering: only merge files containing specified keywords
- Built-in default ignore rules for common binary/build artifact extensions and directories
- Auto-tags code blocks with the correct language based on file extension (for Markdown rendering)

## Build

```bash
go build -o code-merger main.go
```

## Usage

```bash
code-merger [options]
```

### Options

| Flag | Description | Default |
|---|---|---|
| `-i` | Input project directory, can be specified multiple times | `.` |
| `-o` | Output file path | `project_context.txt` |
| `-f` | Specify individual file path(s) to process, bypassing directory traversal (repeatable) | - |
| `-exclude` | Directories to exclude, comma-separated (e.g. `-exclude 'log,core'`) | - |
| `-include` | Directories to forcibly include, comma-separated (takes priority over exclude and default ignore list) | - |
| `-exclude-file` | Files to exclude, comma-separated | - |
| `-include-file` | Files to forcibly include, comma-separated | - |
| `-m` | Match keywords, repeatable; a file is kept if it contains any one keyword | - |

### Examples

Merge all source files in the current directory:

```bash
code-merger -o context.txt
```

Merge multiple directories, exclude `log` and `test` directories, and only keep files containing the `TODO` keyword:

```bash
code-merger -i ./cmd -i ./internal -exclude 'log,test' -m TODO -o context.txt
```

Process only specific files:

```bash
code-merger -f main.go -f internal/router.go -o context.txt
```

Force-include the `build` directory even though it's ignored by default:

```bash
code-merger -i . -include build -o context.txt
```

## Two Modes of Operation

- **Mode A (Specific Files Mode)**: Triggered when `-f` is passed. Processes the given file list directly, without directory traversal and without generating a directory tree.
- **Mode B (Directory Traversal Mode)**: The default mode. First generates a tree structure for each `-i` directory, then recursively merges source files that match the filtering rules.

## Default Ignore Rules

**Directories** (matched by exact name):
`.git` `.github` `node_modules` `vendor` `.idea` `.vscode` `dist` `build`

**File extensions**:
`.exe` `.bmp` `.png` `.jpg` `.jpeg` `.webp` `.ico` `.gif` `.zip` `.tar` `.gz` `.db`

`-include` / `-include-file` rules have the highest priority and can override both the default ignore rules and `-exclude` rules.

## Matching Rules (shared logic for include/exclude)

A rule string matches a path if any of the following conditions hold:

1. The directory/file name exactly equals the rule
2. The relative path exactly equals the rule
3. The relative path starts with `rule/` (subdirectory match)
4. The relative path ends with `/rule` or contains a `/rule/` segment (match at any depth)

## Output Format

The generated file contains:

```
# Project Source Code Context

## 1. Project Structure
```text
[project-name]
├── main.go
└── internal/
    └── ...
```

## 2. Source Code Details

### File: project-name/main.go
```go
...file content...
```
```

## License

GPL-3.0
