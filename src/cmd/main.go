// File: src/cmd/main.go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func main() {
	// Define command-line flags with default values
	repoPath := flag.String("repo", ".", "Path to your local repository")
	outputFile := flag.String("output", "", "Output file name (optional)")
	logLevel := flag.String("log-level", "info", "Set the logging level (debug, info, warn, error)")
	flag.Parse()

	// Set the default output file name if not provided
	if *outputFile == "" {
		*outputFile = fmt.Sprintf("combined_repo_%s_%s.txt", runtime.GOOS, time.Now().Format("20060102T150405"))
	}

	// Configure logger based on log level
	var level slog.Level
	switch strings.ToLower(*logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	logger.Info("Starting Colligo", "repoPath", *repoPath, "outputFile", *outputFile)

	// Normalize repo path
	normalizedRepoPath, err := filepath.Abs(filepath.Clean(*repoPath))
	if err != nil {
		logger.Error("Failed to normalize repository path", "repoPath", *repoPath, "error", err)
		os.Exit(1)
	}
	*repoPath = normalizedRepoPath

	// Open the output file for writing
	outFile, err := os.Create(*outputFile)
	if err != nil {
		logger.Error("Error creating output file", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			logger.Error("Error closing output file", "error", err)
		}
	}()

	writer := bufio.NewWriter(outFile)

	// Walk through the repository directory
	err = filepath.WalkDir(*repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			logger.Error("Error accessing path", "path", path, "error", err)
			return err
		}

		// Get the relative path
		relativePath, err := filepath.Rel(*repoPath, path)
		if err != nil {
			logger.Error("Error getting relative path", "base", *repoPath, "target", path, "error", err)
			return err
		}

		// Normalize and evaluate symbolic links
		evaluatedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			logger.Error("Failed to evaluate symbolic link", "path", path, "error", err)
			return err
		}

		normalizedPath, err := filepath.Abs(filepath.Clean(evaluatedPath))
		if err != nil {
			logger.Error("Failed to normalize path", "path", path, "error", err)
			return err
		}
		path = normalizedPath

		// Skip the output file if it's within the repo directory
		if relativePath == *outputFile {
			return nil
		}

		// Exclude hidden files and directories, but include .github
		if d.IsDir() {
			if isHidden(d.Name()) && d.Name() != ".github" {
				return filepath.SkipDir
			}
			return nil
		} else {
			if isHidden(d.Name()) {
				return nil
			}
		}

		// Write the file content to the output file
		err = writeFileContent(logger, writer, path, relativePath)
		if err != nil {
			logger.Error("Error processing file", "file", path, "error", err)
		}

		return nil
	})

	if err != nil {
		logger.Error("Error walking the path", "repoPath", *repoPath, "error", err)
		os.Exit(1)
	}

	// Flush the buffer to ensure all content is written
	if err = writer.Flush(); err != nil {
		logger.Error("Error flushing writer", "error", err)
		os.Exit(1)
	}

	logger.Info("Successfully combined files", "outputFile", *outputFile)
}

// Helper function to determine if a file or directory is hidden
func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// Helper function to write the content of a file to the writer
func writeFileContent(logger *slog.Logger, writer *bufio.Writer, filePath string, relativePath string) error {
	// Write the header
	_, err := writer.WriteString(fmt.Sprintf("\n\n# BEGIN FILE: %s\n\n", relativePath))
	if err != nil {
		logger.Error("Error writing header", "file", relativePath, "error", err)
		return err
	}

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Error opening file", "file", filePath, "error", err)
		// Write error message to the output file
		_, writeErr := writer.WriteString(fmt.Sprintf("# Error reading %s: %v\n", relativePath, err))
		if writeErr != nil {
			logger.Error("Error writing error message to output", "file", relativePath, "error", writeErr)
			return writeErr
		}
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("Error closing input file", "file", filePath, "error", err)
		}
	}()

	// Copy the file content to the writer
	_, err = io.Copy(writer, file)
	if err != nil {
		logger.Error("Error copying file content", "file", filePath, "error", err)
		return err
	}

	// Write the footer
	_, err = writer.WriteString(fmt.Sprintf("\n\n# END FILE: %s\n\n", relativePath))
	if err != nil {
		logger.Error("Error writing footer", "file", relativePath, "error", err)
	}
	return err
}
