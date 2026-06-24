# code2prompt

code2prompt is a command-line tool that automatically merges source code, configuration files, and documentation into a single context file. It is designed to help developers quickly prepare an entire project for AI models such as OpenAI GPT, Anthropic Claude, Google Gemini, DeepSeek DeepSeek, and Alibaba Cloud Qwen for code analysis, refactoring, debugging, documentation generation, and project understanding.

# Usage

```bash
code2prompt [options]
```

# Command-Line Options

| Option | Default | Description |
|--------|--------|--------|
| -i string | . | Specify the project root directory to scan |
| -o string | project_context.txt | Specify the output context file |
| -f value | Empty | Specify one or more files to process (can be used multiple times). When provided, full project scanning via -i is skipped |
| -m value | Empty | Filter files by keyword (can be used multiple times). Files containing any specified keyword will be included |

# Basic Examples

Scan the current directory:

```bash
code2prompt
```

Equivalent to:

```bash
code2prompt -i .
```

# Specify a Project Directory

```bash
code2prompt -i ./my-project
```

Output：

```text
project_context.txt
```

# Specify an Output File

```bash
code2prompt -i ./my-project -o output.txt
```

# File Filtering

Extract only Dart files:

```bash
code2prompt -i . -f "*.dart"
```

Extract both Go and Dart files:

```bash
code2prompt -i . -f "*.go" -f "*.dart"
```

Extract specific files:

```bash
code2prompt -f pubspec.yaml -f README.md
```

Extract multiple key files:

```bash
code2prompt -f main.go -f config.yaml -f Dockerfile
```

# Content Keyword Filtering

Extract only files containing the keyword riverpod:

```bash
code2prompt -i . -m riverpod
```

Match multiple keywords:

```bash
code2prompt -i . -m riverpod -m flutter_riverpod
```

Any file containing at least one of the specified keywords will be included in the output.

# Combining File and Content Filters

Extract Riverpod-related code from all Dart files:

```bash
code2prompt -i . -f "*.dart" -m riverpod
```
