package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// stringSlice defines a custom type to collect multiple repeated command-line flags.
type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Map of directories or file extensions to ignore during processing.
var ignoredDirs = map[string]bool{
	".git":         true,
	".github":      true,
	"node_modules": true,
	"vendor":       true,
	".idea":        true,
	".vscode":      true,
	"dist":         true,
	"build":        true,
}

var ignoredExtensions = map[string]bool{
	".exe":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".zip":  true,
	".tar":  true,
	".gz":   true,
	".db":   true,
}

func main() {
	// 1. Define command-line arguments
	var targetImports stringSlice
	flag.Var(&targetImports, "import", "Specify the import string to match (can be used multiple times for multiple targets)")
	
	// Customize the default usage helper to accommodate the optional project directory argument
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [project_directory]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// 2. Resolve the target project directory (defaults to current directory ".")
	inputDir := "."
	args := flag.Args()
	if len(args) > 0 {
		inputDir = args[0]
	}

	absPath, err := filepath.Abs(inputDir)
	if err != nil {
		fmt.Printf("Failed to resolve absolute path: %v\n", err)
		return
	}

	outputFile := "project_context.txt"
	fmt.Printf("Analyzing project: %s\n", absPath)
	if len(targetImports) > 0 {
		fmt.Printf("Filter mode active. Matching files containing these imports: %v\n", targetImports)
	}

	out, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		return
	}
	defer out.Close()

	// 3. Write metadata header
	out.WriteString("# Project Source Code Context\n\n")
	out.WriteString("This is an automatically generated project context file intended for AI analysis.\n\n")

	// 4. Generate and write the project directory tree
	out.WriteString("## 1. Project Structure\n```text\n")
	generateTree(absPath, "", out)
	out.WriteString("\n```\n\n")

	// 5. Consolidate source file contents
	out.WriteString("## 2. Source Code Details\n\n")
	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories completely
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip binary/asset extensions and prevent the tool from reading its own output
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ignoredExtensions[ext] || d.Name() == outputFile {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			// Skip files that cannot be read
			return nil
		}

		// ==================== Core Filter Logic: Import Flag Matching ====================
		if len(targetImports) > 0 {
			matched := false
			fileStr := string(content)
			for _, imp := range targetImports {
				if strings.Contains(fileStr, imp) {
					matched = true
					break
				}
			}
			// If the file does not contain any of the specified strings, skip it
			if !matched {
				return nil
			}
		}
		// =================================================================================

		// Calculate relative path for output mapping
		relPath, _ := filepath.Rel(absPath, path)

		// Append the formatted file contents to the output
		out.WriteString(fmt.Sprintf("### File: %s\n", relPath))
		out.WriteString(fmt.Sprintf("```%s\n", getLanguageByExt(ext)))
		out.Write(content)
		out.WriteString("\n```\n\n")

		fmt.Printf("Consolidated: %s\n", relPath)
		return nil
	})

	if err != nil {
		fmt.Printf("Error encountered while traversing files: %v\n", err)
		return
	}

	fmt.Printf("\nSuccess! All consolidated code written to: %s\n", outputFile)
}

// Helper function: Recursively generates a text-based directory tree
func generateTree(root, indent string, out *os.File) {
	files, err := os.ReadDir(root)
	if err != nil {
		return
	}

	var filteredFiles []fs.DirEntry
	for _, f := range files {
		if f.IsDir() && ignoredDirs[f.Name()] {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if !f.IsDir() && ignoredExtensions[ext] {
			continue
		}
		filteredFiles = append(filteredFiles, f)
	}

	for i, f := range filteredFiles {
		isLast := i == len(filteredFiles)-1
		marker := "├── "
		if isLast {
			marker = "└── "
		}

		out.WriteString(fmt.Sprintf("%s%s%s\n", indent, marker, f.Name()))

		if f.IsDir() {
			nextIndent := indent + "│   "
			if isLast {
				nextIndent = indent + "    "
			}
			generateTree(filepath.Join(root, f.Name()), nextIndent, out)
		}
	}
}

// Helper function: Maps file extensions to standard Markdown language identifiers for syntax highlighting
func getLanguageByExt(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".sh":
		return "bash"
	default:
		return "text"
	}
}
