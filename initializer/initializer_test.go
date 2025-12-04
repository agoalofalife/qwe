package initializer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	er "github.com/mainak55512/qwe/qwerror"
	utl "github.com/mainak55512/qwe/qweutils"
	tr "github.com/mainak55512/qwe/tracker"
)

// setupTestDir creates a temp directory and changes to it.
// Returns the temp directory path and a cleanup function.
func setupTestDir(t *testing.T) (tempDirPath string, cleanup func()) {
	t.Helper()

	// Save original working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Create temp directory
	tempDirPath, err = os.MkdirTemp("", "qwe-init-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDirPath); err != nil {
		os.RemoveAll(tempDirPath)
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Return cleanup function
	cleanup = func() {
		os.Chdir(originalDir)
		os.RemoveAll(tempDirPath)
	}

	return tempDirPath, cleanup
}

// TestInit_SuccessfulInitialization tests that Init successfully creates all required files and directories
func TestInit_SuccessfulInitialization(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()

	err := Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	qwePath := filepath.Join(tempDir, ".qwe")
	if _, err := os.Stat(qwePath); os.IsNotExist(err) {
		t.Error(".qwe directory was not created")
	}

	// Verify _object directory exists
	objectPath := filepath.Join(tempDir, ".qwe", "_object")
	if info, err := os.Stat(objectPath); os.IsNotExist(err) {
		t.Error(".qwe/_object directory was not created")
	} else if !info.IsDir() {
		t.Error(".qwe/_object is not a directory")
	}

	// Verify _tracker.qwe file exists
	trackerPath := filepath.Join(tempDir, ".qwe", "_tracker.qwe")
	if _, err := os.Stat(trackerPath); os.IsNotExist(err) {
		t.Error(".qwe/_tracker.qwe file was not created")
	}

	// Verify _group_tracker.qwe file exists
	groupTrackerPath := filepath.Join(tempDir, ".qwe", "_group_tracker.qwe")
	if _, err := os.Stat(groupTrackerPath); os.IsNotExist(err) {
		t.Error(".qwe/_group_tracker.qwe file was not created")
	}

	_, _, err = tr.GetTracker(tr.FileTrackerType)

	if err != nil {
		t.Fatalf("failed to read tracker file: %v", err)
	}
}

// TestInit_RepoAlreadyInitialized tests that Init returns error when repository is already initialized
func TestInit_RepoAlreadyInitialized(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// First initialization should succeed
	err := Init()
	if err != nil {
		t.Fatalf("first Init() call failed: %v", err)
	}

	// Second initialization should fail with RepoAlreadyInit error
	err = Init()
	if err == nil {
		t.Fatal("expected error when initializing already initialized repository, got nil")
	}

	// Verify it's the specific RepoAlreadyInit error
	if !errors.Is(err, er.RepoAlreadyInit) {
		if err != er.RepoAlreadyInit {
			t.Errorf("expected RepoAlreadyInit error, got: %v", err)
		}
	}
}

// TestGroupInit_SuccessfulInitialization tests successful group initialization
func TestGroupInit_SuccessfulInitialization(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Initialize qwe repository first
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Initialize a group
	groupName := "test-group"
	err := GroupInit(groupName)
	if err != nil {
		t.Fatalf("GroupInit() failed: %v", err)
	}

	// Verify group was added to group tracker
	_, groupTracker, err := tr.GetTracker(tr.GroupTrackerType)
	if err != nil {
		t.Fatalf("failed to get group tracker: %v", err)
	}

	groupID := utl.Hasher(groupName)
	group, exists := groupTracker[groupID]
	if !exists {
		t.Fatal("group was not added to group tracker")
	}

	// Verify group details
	if group.GroupName != groupName {
		t.Errorf("expected group name %s, got %s", groupName, group.GroupName)
	}

	if group.Current == "" {
		t.Error("group current version is empty")
	}

	if len(group.VersionOrder) != 1 {
		t.Errorf("expected 1 version in version order, got %d", len(group.VersionOrder))
	}

	if len(group.Versions) != 1 {
		t.Errorf("expected 1 version, got %d", len(group.Versions))
	}

	// Verify initial version details
	initialVersion := group.Versions[group.Current]
	if initialVersion.CommitMessage != "Initial Tracking" {
		t.Errorf("expected commit message 'Initial Tracking', got %s", initialVersion.CommitMessage)
	}

	if len(initialVersion.Files) != 0 {
		t.Errorf("expected no files in initial version, got %d", len(initialVersion.Files))
	}
}

// TestGroupInit_RepoNotInitialized tests that GroupInit fails when repository is not initialized
func TestGroupInit_RepoNotInitialized(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Try to initialize group without initializing repository first
	err := GroupInit("test-group")
	if err == nil {
		t.Fatal("expected error when initializing group without repository, got nil")
	}

	if !errors.Is(err, er.RepoNotFound) {
		if err != er.RepoNotFound {
			t.Errorf("expected RepoNotFound error, got: %v", err)
		}
	}
}

// TestGroupInit_GroupAlreadyTracked tests that GroupInit fails when group is already tracked
func TestGroupInit_GroupAlreadyTracked(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Initialize qwe repository
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	groupName := "test-group"

	// First group initialization should succeed
	if err := GroupInit(groupName); err != nil {
		t.Fatalf("first GroupInit() failed: %v", err)
	}

	// Second initialization of same group should fail
	err := GroupInit(groupName)
	if err == nil {
		t.Fatal("expected error when initializing already tracked group, got nil")
	}

	// Verify it's the specific GrpAlreadyTracked error
	if !errors.Is(err, er.GrpAlreadyTracked) {
		if err != er.GrpAlreadyTracked {
			t.Errorf("expected GrpAlreadyTracked error, got: %v", err)
		}
	}
}

// TestGroupInit_MultipleGroups tests initializing multiple different groups
func TestGroupInit_MultipleGroups(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Initialize qwe repository
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Initialize multiple groups
	groups := []string{"group1", "group2", "group3"}
	for _, groupName := range groups {
		if err := GroupInit(groupName); err != nil {
			t.Fatalf("GroupInit(%s) failed: %v", groupName, err)
		}
	}

	// Verify all groups exist in tracker
	_, groupTracker, err := tr.GetTracker(tr.GroupTrackerType)
	if err != nil {
		t.Fatalf("failed to get group tracker: %v", err)
	}

	if len(groupTracker) != len(groups) {
		t.Errorf("expected %d groups in tracker, got %d", len(groups), len(groupTracker))
	}

	// Verify each group
	for _, groupName := range groups {
		groupID := utl.Hasher(groupName)
		group, exists := groupTracker[groupID]
		if !exists {
			t.Errorf("group %s not found in tracker", groupName)
			continue
		}

		if group.GroupName != groupName {
			t.Errorf("expected group name %s, got %s", groupName, group.GroupName)
		}
	}
}

// TestGroupInit_ValidatesGroupName tests group initialization with various group names
func TestGroupInit_ValidatesGroupName(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Initialize qwe repository
	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	tests := []struct {
		name      string
		groupName string
		shouldErr bool
	}{
		{
			name:      "simple group name",
			groupName: "mygroup",
			shouldErr: false,
		},
		{
			name:      "group with hyphen",
			groupName: "my-group",
			shouldErr: false,
		},
		{
			name:      "group with underscore",
			groupName: "my_group",
			shouldErr: false,
		},
		{
			name:      "group with numbers",
			groupName: "group123",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GroupInit(tt.groupName)
			if tt.shouldErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
