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

// Default excluded directory names (still keeping pure name matching as a quick base filter)
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
	".bmp":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".ico": true,
	".gif":  true,
	".zip":  true,
	".tar":  true,
	".gz":   true,
	".db":   true,
}

func main() {
	var inputDirs stringSlice
	var excludeDirs stringSlice
	var outputFilePath string
	var specificFiles stringSlice
	var matchKeywords stringSlice

	flag.Var(&inputDirs, "i", "Specify the input project directory (can be used multiple times; defaults to '.' if none provided)")
	flag.Var(&excludeDirs, "e", "Specify directories to exclude/ignore (e.g., -e 'internal/winipcfg' or -e '.git')")
	flag.StringVar(&outputFilePath, "o", "project_context.txt", "Specify the output file path")
	flag.Var(&specificFiles, "f", "Specify single/multiple file paths to process (can be used multiple times; will bypass directory traversal)")
	flag.Var(&matchKeywords, "m", "Match keywords for content filtering (can be used multiple times; matches if a file contains any keyword)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(inputDirs) == 0 {
		inputDirs = append(inputDirs, ".")
	}

	// Process user-defined exclusion paths:
	// Uniformly convert backslashes \ to forward slashes / for cross-platform matching, and trim leading/trailing slashes
	var userExcludePaths []string
	for _, dir := range excludeDirs {
		cleanDir := strings.ReplaceAll(dir, "\\", "/")
		cleanDir = strings.Trim(cleanDir, "/")
		if cleanDir != "" {
			userExcludePaths = append(userExcludePaths, cleanDir)
		}
	}

	out, err := os.Create(outputFilePath)
	if err != nil {
		fmt.Printf("Unable to create output file: %v\n", err)
		return
	}
	defer out.Close()

	out.WriteString("# Project Source Code Context\n\n")
	out.WriteString("This is an automatically generated project context file intended for AI analysis.\n\n")

	if len(specificFiles) > 0 {
		// --- Mode A: Specific Files Mode ---
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
		
		// 3.1 Generate directory tree structure
		out.WriteString("## 1. Project Structure\n```text\n")
		for _, inDir := range inputDirs {
			absInputDir, err := filepath.Abs(inDir)
			if err != nil {
				fmt.Printf("Unable to resolve the absolute path of input directory [%s]: %v\n", inDir, err)
				continue
			}
			fmt.Printf("Analyzing project directory structure: %s\n", absInputDir)
			out.WriteString(fmt.Sprintf("[%s]\n", filepath.Base(absInputDir)))
			generateTree(absInputDir, absInputDir, "", userExcludePaths, out)
			out.WriteString("\n")
		}
		out.WriteString("```\n\n")

		if len(matchKeywords) > 0 {
			fmt.Printf("Filter activated. Only merging files containing keywords: %v\n", matchKeywords)
		}

		out.WriteString("## 2. Source Code Details\n\n")

		// 3.2 Source code merging phase
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

				if d.IsDir() {
					// Calculate the current directory's path relative to the input root directory
					relDir, _ := filepath.Rel(absInputDir, path)
					if shouldIgnore(d.Name(), relDir, userExcludePaths) {
						return filepath.SkipDir
					}
					return nil
				}

				ext := strings.ToLower(filepath.Ext(d.Name()))
				if ignoredExtensions[ext] || d.Name() == filepath.Base(outputFilePath) {
					return nil
				}

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

// Determines whether a directory should be excluded
func shouldIgnore(dirName, relPath string, userExcludePaths []string) bool {
	// 1. Matches default global base filters (e.g., .git, node_modules)
	if defaultIgnoredDirs[dirName] {
		return true
	}

	// Uniformly convert the current path to forward slashes for comparison
	standardRelPath := strings.ReplaceAll(relPath, "\\", "/")

	// 2. Check if it matches rules specified by the user via -e
	for _, exclude := range userExcludePaths {
		// Exact match (e.g., "internal/winipcfg") or current path starts with the exclusion path (handling its subdirectories)
		if standardRelPath == exclude || strings.HasPrefix(standardRelPath, exclude+"/") {
			return true
		}
	}
	return false
}

func processFile(path, baseDir, outputFilePath string, keywords stringSlice, out *os.File) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

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
			return
		}
	}

	relPath, err := filepath.Rel(baseDir, path)
	if err != nil || strings.HasPrefix(relPath, "..") {
		relPath = filepath.Base(path)
	} else {
		relPath = filepath.Join(filepath.Base(baseDir), relPath)
	}

	ext := strings.ToLower(filepath.Ext(path))

	out.WriteString(fmt.Sprintf("### File: %s\n", relPath))
	out.WriteString(fmt.Sprintf("```%s\n", getLanguageByExt(ext)))
	out.Write(content)
	out.WriteString("\n```\n\n")

	fmt.Printf("Merged: %s\n", relPath)
}

// Helper function: Generates a directory tree
func generateTree(baseDir, currentRoot, indent string, userExcludePaths []string, out *os.File) {
	files, err := os.ReadDir(currentRoot)
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

		fullPath := filepath.Join(currentRoot, f.Name())
		relPath, _ := filepath.Rel(baseDir, fullPath)

		isIgnoredDir := f.IsDir() && shouldIgnore(f.Name(), relPath, userExcludePaths)

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
			generateTree(baseDir, fullPath, nextIndent, userExcludePaths, out)
		}
	}
}

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
