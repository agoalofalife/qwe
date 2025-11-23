package diff

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	in "github.com/mainak55512/qwe/initializer"
	er "github.com/mainak55512/qwe/qwerror"
)

func TestDiffArgumentValidation(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		commitID1   string
		commitID2   string
		expectError bool
	}{
		{
			name:        "both empty - valid",
			filePath:    "test.txt",
			commitID1:   "",
			commitID2:   "",
			expectError: false,
		},
		{
			name:        "both non-empty - valid",
			filePath:    "test.txt",
			commitID1:   "1",
			commitID2:   "2",
			expectError: false,
		},
		{
			name:        "first empty second non-empty - invalid",
			filePath:    "test.txt",
			commitID1:   "",
			commitID2:   "1",
			expectError: true,
		},
		{
			name:        "first non-empty second empty - invalid",
			filePath:    "test.txt",
			commitID1:   "1",
			commitID2:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Diff(tt.filePath, tt.commitID1, tt.commitID2)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for mismatched arguments, got nil")
				} else if err.Error() != "Argument number missmatch" {
					t.Errorf("expected 'Argument number missmatch' error, got: %v", err)
				}
			}
		})
	}
}

func TestDiffFileNotTracked(t *testing.T) {
	tempDirPath, cleanup := initQwe(t)
	defer cleanup()

	// Create test file in temp directory
	testFile := filepath.Join(tempDirPath, "untracked.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Try to diff an untracked file (should fail with FileNotTracked)
	err := Diff(testFile, "", "")

	// Verify we get the specific FileNotTracked error
	if err == nil {
		t.Error("expected FileNotTracked error, got nil")
	} else if !errors.Is(err, er.FileNotTracked) {
		if err != er.FileNotTracked {
			t.Errorf("expected FileNotTracked error, got: %v", err)
		}
	}
}

// initQwe creates a temp directory, initializes qwe repository, and changes to that directory.
// Returns the temp directory path and a cleanup function.
// The cleanup function restores the original directory and removes the temp directory.
//
// Usage:
//
//	tempDir, cleanup := initQwe(t)
//	defer cleanup()
func initQwe(t *testing.T) (tempDirPath string, cleanup func()) {
	t.Helper()

	// Save original working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Create temp directory
	tempDirPath, err = os.MkdirTemp("", "qwe-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDirPath); err != nil {
		os.RemoveAll(tempDirPath)
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Initialize qwe repository in temp directory
	if err := in.Init(); err != nil {
		os.Chdir(originalDir)
		os.RemoveAll(tempDirPath)
		t.Fatalf("failed to initialize qwe repository: %v", err)
	}

	// Return cleanup function that will be called with defer
	cleanup = func() {
		os.Chdir(originalDir)
		os.RemoveAll(tempDirPath)
	}

	return tempDirPath, cleanup
}
