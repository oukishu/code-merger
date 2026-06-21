package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Define folders or file extensions that need to be ignored
var ignoredDirs = map[string]bool{
	".git":         true,
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
	// Default to processing the current directory and outputting to project_context.txt
	inputDir := "."
	outputFile := "project_context.txt"

	if len(os.Args) > 1 {
		inputDir = os.Args[1]
	}

	absPath, err := filepath.Abs(inputDir)
	if err != nil {
		fmt.Printf("Failed to get absolute path: %v\n", err)
		return
	}

	fmt.Printf("Analyzing project: %s\n", absPath)

	out, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		return
	}
	defer out.Close()

	// 1. Write header information
	out.WriteString("# Consolidated Project Source Code Context\n\n")
	out.WriteString("This is an automatically generated project code context file for AI analysis.\n\n")

	// 2. Generate and write the project directory tree
	out.WriteString("## 1. Project Directory Structure\n```text\n")
	generateTree(absPath, "", out)
	out.WriteString("\n```\n\n")

	// 3. Consolidate file contents
	out.WriteString("## 2. Source Code Details\n\n")
	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check if the directory should be ignored
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if the file extension should be ignored, and avoid reading the output file itself
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ignoredExtensions[ext] || d.Name() == outputFile {
			return nil
		}

		// Calculate relative path for easier AI location tracking
		relPath, _ := filepath.Rel(absPath, path)

		// Read file contents
		content, err := os.ReadFile(path)
		if err != nil {
			// Skip files that cannot be read (e.g., binary files or lack of permissions)
			return nil
		}

		// Write to the output file, wrapped in Markdown formatting
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

	fmt.Printf("\n Success! All code has been consolidated into: %s\n", outputFile)
}

// Helper function: Generate directory tree
func generateTree(root, indent string, out *os.File) {
	files, err := os.ReadDir(root)
	if err != nil {
		return
	}

	// Filter out ignored directories
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

// Helper function: Get Markdown language syntax highlighting tag by file extension
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
