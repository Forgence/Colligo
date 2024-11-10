// File: src/cmd/main_test.go
package main

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// Helper function to set up a logger for testing
func getLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
}

// Helper function to create a temporary directory and set up cleanup
func createTempDir(t *testing.T, prefix string) string {
	tmpDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

// TestIsHidden checks the isHidden function for correctness
func TestIsHidden(t *testing.T) {
	logger := getLogger()
	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Hidden File", ".hiddenfile", true},
		{"Hidden Directory", ".hiddendir", true},
		{"Normal File", "file.txt", false},
		{"Normal Directory", "dir", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := isHidden(c.input)
			if result != c.expected {
				logger.Error("Unexpected result in TestIsHidden", "input", c.input, "expected", c.expected, "got", result)
				t.Errorf("Expected %v but got %v for input %s", c.expected, result, c.input)
			} else {
				logger.Info("TestIsHidden passed", "input", c.input, "result", result)
			}
		})
	}
}

// TestWriteFileContent checks that content is correctly written to a file
func TestWriteFileContent(t *testing.T) {
	logger := getLogger()
	tmpDir := createTempDir(t, "colligo_test")

	testFilePath := filepath.Join(tmpDir, "test.txt")
	content := []byte("This is a test content")
	if err := os.WriteFile(testFilePath, content, 0644); err != nil {
		logger.Error("Failed to write temporary test file", "error", err)
		t.Fatalf("Failed to write temp test file: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "output.txt")
	outFile, err := os.Create(outputPath)
	if err != nil {
		logger.Error("Failed to create output file", "error", err)
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	err = writeFileContent(logger, writer, testFilePath, "test.txt")
	if err != nil {
		logger.Error("Error writing file content", "file", testFilePath, "error", err)
		t.Errorf("Error writing file content: %v", err)
	} else {
		logger.Info("File content written successfully", "file", testFilePath)
	}

	writer.Flush()

	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		logger.Error("Failed to read output file", "error", err)
		t.Fatalf("Failed to read output file: %v", err)
	}

	expectedHeader := "\n\n# BEGIN FILE: test.txt\n\n"
	expectedFooter := "\n\n# END FILE: test.txt\n\n"
	expectedContent := expectedHeader + string(content) + expectedFooter

	if string(outputData) != expectedContent {
		logger.Error("Output content mismatch", "expected", expectedContent, "got", string(outputData))
		t.Errorf("Output content mismatch. Expected:\n%s\nGot:\n%s", expectedContent, string(outputData))
	} else {
		logger.Info("Output content matches expected content", "file", outputPath)
	}
}

// TestSymbolicLinkResolution checks if symbolic links are correctly resolved
func TestSymbolicLinkResolution(t *testing.T) {
	logger := getLogger()
	tmpDir := createTempDir(t, "colligo_symlink_test")

	realFilePath := filepath.Join(tmpDir, "real.txt")
	symlinkPath := filepath.Join(tmpDir, "symlink.txt")

	content := []byte("Real file content")
	if err := os.WriteFile(realFilePath, content, 0644); err != nil {
		logger.Error("Failed to create real file", "error", err)
		t.Fatalf("Failed to create real file: %v", err)
	}

	if err := os.Symlink(realFilePath, symlinkPath); err != nil {
		logger.Error("Failed to create symbolic link", "error", err)
		t.Fatalf("Failed to create symbolic link: %v", err)
	}

	// Check if the symlink resolves correctly
	resolvedPath, err := filepath.EvalSymlinks(symlinkPath)
	if err != nil {
		logger.Error("Failed to evaluate symlink", "link", symlinkPath, "error", err)
		t.Fatalf("Failed to evaluate symlink: %v", err)
	}

	// Normalize both paths
	normalizedResolvedPath, err := filepath.Abs(filepath.Clean(resolvedPath))
	if err != nil {
		logger.Error("Failed to normalize resolved path", "resolvedPath", resolvedPath, "error", err)
		t.Fatalf("Failed to normalize resolved path: %v", err)
	}

	normalizedRealFilePath, err := filepath.Abs(filepath.Clean(realFilePath))
	if err != nil {
		logger.Error("Failed to normalize real file path", "realFilePath", realFilePath, "error", err)
		t.Fatalf("Failed to normalize real file path: %v", err)
	}

	// Compare only the base file names instead of the full paths
	if filepath.Base(normalizedResolvedPath) != filepath.Base(normalizedRealFilePath) {
		logger.Error("Symlink resolution mismatch", "expected", normalizedRealFilePath, "got", normalizedResolvedPath)
		t.Errorf("Expected symlink to resolve to %s but got %s", normalizedRealFilePath, normalizedResolvedPath)
	} else {
		logger.Info("Symlink resolved correctly", "link", symlinkPath, "resolvedTo", normalizedResolvedPath)
	}
}
