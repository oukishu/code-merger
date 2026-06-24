package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// stringSlice defines a custom type used to collect multiple recurring command-line flags.
type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Directories or file extensions to ignore by default
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
	// 1. Define command-line parameters
	var inputDir string
	var outputFilePath string
	var specificFiles stringSlice
	var matchKeywords stringSlice

	flag.StringVar(&inputDir, "i", ".", "Specify the input project directory")
	flag.StringVar(&outputFilePath, "o", "project_context.txt", "Specify the output file path")
	flag.Var(&specificFiles, "f", "Specify single/multiple file paths to process (can be used multiple times; will bypass -i directory traversal if specified)")
	flag.Var(&matchKeywords, "m", "Match keywords for content filtering (can be used multiple times; matches if a file contains any keyword)")

	// Custom Usage help information
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// 2. Resolve absolute paths
	absInputDir, err := filepath.Abs(inputDir)
	if err != nil {
		fmt.Printf("Unable to resolve the absolute path of the input directory: %v\n", err)
		return
	}

	out, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Printf("Unable to create output file: %v\n", err)
		return
	}
	defer out.Close()

	// 3. Write metadata header content
	out.WriteString("# Project Source Code Context\n\n")
	out.WriteString("This is an automatically generated project context file intended for AI analysis.\n\n")

	// 4. Generate and write project directory tree (only generated when not in specific file mode)
	if len(specificFiles) == 0 {
		fmt.Printf("Analyzing project directory: %s\n", absInputDir)
		out.WriteString("## 1. Project Structure\n```text\n")
		generateTree(absInputDir, "", out)
		out.WriteString("\n```\n\n")
	} else {
		fmt.Printf("Processing %d specified file(s)...\n", len(specificFiles))
	}

	if len(matchKeywords) > 0 {
		fmt.Printf("Filter activated. Only merging files containing the following keywords: %v\n", matchKeywords)
	}

	out.WriteString("## 2. Source Code Details\n\n")

	// 5. Core processing logic: Distinguish between [Specific Files Mode] and [Directory Traversal Mode]
	if len(specificFiles) > 0 {
		// --- Mode A: Process specified files ---
		for _, fPath := range specificFiles {
			absFilePath, err := filepath.Abs(fPath)
			if err != nil {
				fmt.Printf("Unable to resolve file path: %s, error: %v\n", fPath, err)
				continue
			}
			
			// Check if file exists
			info, err := os.Stat(absFilePath)
			if err != nil || info.IsDir() {
				fmt.Printf("Skipping invalid file path: %s\n", fPath)
				continue
			}

			// Process the file
			processFile(absFilePath, absInputDir, outputFilePath, matchKeywords, out)
		}
	} else {
		// --- Mode B: Traditional directory traversal mode ---
		err = filepath.WalkDir(absInputDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip ignored directories
			if d.IsDir() {
				if ignoredDirs[d.Name()] {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip binary/asset extensions, and prevent the tool from reading its own output
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ignoredExtensions[ext] || d.Name() == filepath.Base(outputFilePath) {
				return nil
			}

			// Process the file
			processFile(path, absInputDir, outputFilePath, matchKeywords, out)
			return nil
		})

		if err != nil {
			fmt.Printf("Encountered an error while traversing files: %v\n", err)
			return
		}
	}

	fmt.Printf("\nSuccess! All merged code has been written to: %s\n", outputFilePath)
}

// Extracted single-file processing logic function (handles keyword filtering and writing)
func processFile(path, baseDir, outputFilePath string, keywords stringSlice, out *os.File) {
	// Read file contents
	content, err := os.ReadFile(path)
	if err != nil {
		return // Skip files that cannot be read
	}

	// Keyword filtering logic (-m)
	if len(keywords) > 0 {
		matched := false
		fileStr := string(content)
		for _, kw := range keywords {
			if strings.Contains(fileStr, kw) {
				matched = true
				break
			}
		}
		if !matched {
			return // If it doesn't contain any keywords, skip it
		}
	}

	// Calculate relative path for display mapping
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil || strings.HasPrefix(relPath, "..") {
		// If the file is not within inputDir (e.g., an external file was passed via -f), directly display absolute path or filename
		relPath = filepath.Base(path)
	}

	ext := strings.ToLower(filepath.Ext(path))

	// Write to the output file
	out.WriteString(fmt.Sprintf("### File: %s\n", relPath))
	out.WriteString(fmt.Sprintf("```%s\n", getLanguageByExt(ext)))
	out.Write(content)
	out.WriteString("\n```\n\n")

	fmt.Printf("Merged: %s\n", relPath)
}

// Helper function: Recursively generates a text directory tree
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

// Helper function: Maps extensions to Markdown syntax highlighting identifiers
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
