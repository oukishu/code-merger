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

// Default directories or file extensions to ignore
var defaultIgnoredDirs = map[string]bool{
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
	var inputDirs stringSlice
	var excludeDirs stringSlice
	var outputFilePath string
	var specificFiles stringSlice
	var matchKeywords stringSlice

	flag.Var(&inputDirs, "i", "Specify the input project directory (can be used multiple times; defaults to '.' if none provided)")
	flag.Var(&excludeDirs, "e", "Specify directories to exclude/ignore (can be used multiple times; appends to default list)")
	flag.StringVar(&outputFilePath, "o", "project_context.txt", "Specify the output file path")
	flag.Var(&specificFiles, "f", "Specify single/multiple file paths to process (can be used multiple times; will bypass directory traversal)")
	flag.Var(&matchKeywords, "m", "Match keywords for content filtering (can be used multiple times; matches if a file contains any keyword)")

	// Custom Usage help information
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// If no input directories specified, default to current directory
	if len(inputDirs) == 0 {
		inputDirs = append(inputDirs, ".")
	}

	// Merge default ignored directories with user-specified exclusions
	ignoredDirs := make(map[string]bool)
	for k, v := range defaultIgnoredDirs {
		ignoredDirs[k] = v
	}
	for _, dir := range excludeDirs {
		ignoredDirs[dir] = true
	}

	out, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Printf("Unable to create output file: %v\n", err)
		return
	}
	defer out.Close()

	// 2. Write metadata header content
	out.WriteString("# Project Source Code Context\n\n")
	out.WriteString("This is an automatically generated project context file intended for AI analysis.\n\n")

	// 3. Core processing logic: Distinguish between [Specific Files Mode] and [Directory Traversal Mode]
	if len(specificFiles) > 0 {
		// --- Mode A: Process specified files ---
		fmt.Printf("Processing %d specified file(s)...\n", len(specificFiles))
		out.WriteString("## 1. Source Code Details (Specific Files Mode)\n\n")

		for _, fPath := range specificFiles {
			absFilePath, err := filepath.Abs(fPath)
			if err != nil {
				fmt.Printf("Unable to resolve file path: %s, error: %v\n", fPath, err)
				continue
			}

			info, err := os.Stat(absFilePath)
			if err != nil || info.IsDir() {
				fmt.Printf("Skipping invalid file path: %s\n", fPath)
				continue
			}

			processFile(absFilePath, filepath.Dir(absFilePath), outputFilePath, matchKeywords, out)
		}
	} else {
		// --- Mode B: Multiple directory traversal mode ---
		
		// 3.1 Generate Project Structure for all input directories
		out.WriteString("## 1. Project Structure\n```text\n")
		for _, inDir := range inputDirs {
			absInputDir, err := filepath.Abs(inDir)
			if err != nil {
				fmt.Printf("Unable to resolve the absolute path of input directory [%s]: %v\n", inDir, err)
				continue
			}
			fmt.Printf("Analyzing project directory structure: %s\n", absInputDir)
			out.WriteString(fmt.Sprintf("[%s]\n", filepath.Base(absInputDir)))
			generateTree(absInputDir, "", ignoredDirs, out)
			out.WriteString("\n")
		}
		out.WriteString("```\n\n")

		if len(matchKeywords) > 0 {
			fmt.Printf("Filter activated. Only merging files containing keywords: %v\n", matchKeywords)
		}

		out.WriteString("## 2. Source Code Details\n\n")

		// 3.2 Traverse and process source files for each directory
		for _, inDir := range inputDirs {
			absInputDir, err := filepath.Abs(inDir)
			if err != nil {
				continue
			}

			fmt.Printf("Merging source files from: %s\n", absInputDir)
			err = filepath.WalkDir(absInputDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Skip ignored directories completely during file contents merging
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
				fmt.Printf("Encountered an error while traversing [%s]: %v\n", inDir, err)
			}
		}
	}

	fmt.Printf("\nSuccess! All merged code has been written to: %s\n", outputFilePath)
}

// Extracted single-file processing logic function (handles keyword filtering and writing)
func processFile(path, baseDir, outputFilePath string, keywords stringSlice, out *os.File) {
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

	// [Path Optimization Core]
	// Calculate the file path relative to the current input directory (baseDir) being traversed
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil || strings.HasPrefix(relPath, "..") {
		relPath = filepath.Base(path)
	} else {
		// Prepend the base directory's own last element name (e.g., "A") back onto the front of the path
		relPath = filepath.Join(filepath.Base(baseDir), relPath)
	}

	ext := strings.ToLower(filepath.Ext(path))

	// Write to the output file
	out.WriteString(fmt.Sprintf("### File: %s\n", relPath))
	out.WriteString(fmt.Sprintf("```%s\n", getLanguageByExt(ext)))
	out.Write(content)
	out.WriteString("\n```\n\n")

	fmt.Printf("Merged: %s\n", relPath)
}

// Helper function: Recursively generates a text directory tree (Includes ignored directories in framework skeleton)
func generateTree(root, indent string, ignoredDirs map[string]bool, out *os.File) {
	files, err := os.ReadDir(root)
	if err != nil {
		return
	}

	var filteredFiles []fs.DirEntry
	for _, f := range files {
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

		isIgnoredDir := f.IsDir() && ignoredDirs[f.Name()]

		displayName := f.Name()
		if isIgnoredDir {
			displayName = fmt.Sprintf("%s/ (excluded)", f.Name())
		}

		out.WriteString(fmt.Sprintf("%s%s%s\n", indent, marker, displayName))

		if f.IsDir() && !isIgnoredDir {
			nextIndent := indent + "│   "
			if isLast {
				nextIndent = indent + "    "
			}
			generateTree(filepath.Join(root, f.Name()), nextIndent, ignoredDirs, out)
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
