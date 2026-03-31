package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if ws == nil {
		t.Fatal("New() returned nil workspace")
	}
	if ws.rickDir == "" {
		t.Fatal("rickDir is empty")
	}
}

func TestInitWorkspace(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := ws.InitWorkspace(); err != nil {
		t.Fatalf("InitWorkspace() failed: %v", err)
	}

	// Verify directories were created
	requiredDirs := []string{
		WikiDirName,
		SkillsDirName,
		JobsDirName,
	}

	for _, dir := range requiredDirs {
		path := filepath.Join(ws.rickDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}

	// Verify files were created
	okriPath := filepath.Join(ws.rickDir, OKRFileName)
	if _, err := os.Stat(okriPath); os.IsNotExist(err) {
		t.Error("OKR.md was not created")
	}

	specPath := filepath.Join(ws.rickDir, SpecFileName)
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("SPEC.md was not created")
	}
}

func TestGetJobPath(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name    string
		jobID   string
		wantErr bool
	}{
		{
			name:    "valid job ID",
			jobID:   "job_1",
			wantErr: false,
		},
		{
			name:    "empty job ID",
			jobID:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := ws.GetJobPath(tt.jobID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetJobPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && path == "" {
				t.Error("GetJobPath() returned empty path")
			}
			if !tt.wantErr && !filepath.IsAbs(path) {
				t.Error("GetJobPath() returned relative path")
			}
		})
	}
}

func TestCreateJobStructure(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Initialize workspace first
	if err := ws.InitWorkspace(); err != nil {
		t.Fatalf("InitWorkspace() failed: %v", err)
	}

	jobID := "test_job"
	if err := ws.CreateJobStructure(jobID); err != nil {
		t.Fatalf("CreateJobStructure() failed: %v", err)
	}

	// Verify job structure was created
	requiredDirs := []string{
		PlanDirName,
		DoingDirName,
		LearningDirName,
	}

	jobPath, _ := ws.GetJobPath(jobID)
	for _, dir := range requiredDirs {
		path := filepath.Join(jobPath, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Job subdirectory %s was not created", dir)
		}
	}
}

func TestCreateJobStructureEmptyJobID(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := ws.CreateJobStructure(""); err == nil {
		t.Error("CreateJobStructure() should fail with empty jobID")
	}
}

func TestEnsureDirectories(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := ws.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories() failed: %v", err)
	}

	// Verify directories exist
	requiredDirs := []string{
		WikiDirName,
		SkillsDirName,
		JobsDirName,
	}

	for _, dir := range requiredDirs {
		path := filepath.Join(ws.rickDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
}

func TestEnsureDirectoriesIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Call twice to ensure idempotency
	if err := ws.EnsureDirectories(); err != nil {
		t.Fatalf("First EnsureDirectories() failed: %v", err)
	}

	if err := ws.EnsureDirectories(); err != nil {
		t.Fatalf("Second EnsureDirectories() failed: %v", err)
	}
}

func TestGetRickDir(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create temp directory and change to it
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	rickDir := ws.GetRickDir()
	if rickDir == "" {
		t.Error("GetRickDir() returned empty string")
	}

	// Should be .rick in current directory (tempDir)
	expectedDir := filepath.Join(tempDir, RickDirName)

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	rickDirResolved, _ := filepath.EvalSymlinks(rickDir)
	expectedDirResolved, _ := filepath.EvalSymlinks(expectedDir)

	if rickDirResolved != expectedDirResolved {
		t.Errorf("GetRickDir() = %s, want %s", rickDirResolved, expectedDirResolved)
	}
}

func TestPathConstants(t *testing.T) {
	// Verify path constants are not empty
	constants := map[string]string{
		"RickDirName":     RickDirName,
		"OKRFileName":     OKRFileName,
		"SpecFileName":    SpecFileName,
		"WikiDirName":     WikiDirName,
		"SkillsDirName":   SkillsDirName,
		"JobsDirName":     JobsDirName,
		"PlanDirName":     PlanDirName,
		"DoingDirName":    DoingDirName,
		"LearningDirName": LearningDirName,
	}

	for name, value := range constants {
		if value == "" {
			t.Errorf("Path constant %s is empty", name)
		}
	}
}

func TestPathFunctions(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create temp directory and change to it
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test GetRickDir - should return .rick in current directory
	rickDir, err := GetRickDir()
	if err != nil {
		t.Fatalf("GetRickDir() failed: %v", err)
	}
	if rickDir == "" {
		t.Error("GetRickDir() returned empty string")
	}

	// Test GetJobsDir
	jobsDir, err := GetJobsDir()
	if err != nil {
		t.Fatalf("GetJobsDir() failed: %v", err)
	}
	if jobsDir == "" {
		t.Error("GetJobsDir() returned empty string")
	}

	// Test GetJobDir
	jobDir, err := GetJobDir("test_job")
	if err != nil {
		t.Fatalf("GetJobDir() failed: %v", err)
	}
	if jobDir == "" {
		t.Error("GetJobDir() returned empty string")
	}

	// Test GetJobPlanDir
	planDir, err := GetJobPlanDir("test_job")
	if err != nil {
		t.Fatalf("GetJobPlanDir() failed: %v", err)
	}
	if planDir == "" {
		t.Error("GetJobPlanDir() returned empty string")
	}

	// Test GetJobDoingDir
	doingDir, err := GetJobDoingDir("test_job")
	if err != nil {
		t.Fatalf("GetJobDoingDir() failed: %v", err)
	}
	if doingDir == "" {
		t.Error("GetJobDoingDir() returned empty string")
	}

	// Test GetJobLearningDir
	learningDir, err := GetJobLearningDir("test_job")
	if err != nil {
		t.Fatalf("GetJobLearningDir() failed: %v", err)
	}
	if learningDir == "" {
		t.Error("GetJobLearningDir() returned empty string")
	}
}

func TestInitWorkspaceIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Call InitWorkspace twice
	if err := ws.InitWorkspace(); err != nil {
		t.Fatalf("First InitWorkspace() failed: %v", err)
	}

	if err := ws.InitWorkspace(); err != nil {
		t.Fatalf("Second InitWorkspace() failed: %v", err)
	}

	// Verify files still exist
	okriPath := filepath.Join(ws.rickDir, OKRFileName)
	if _, err := os.Stat(okriPath); os.IsNotExist(err) {
		t.Error("OKR.md was removed")
	}
}

func TestMultipleJobStructures(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := ws.InitWorkspace(); err != nil {
		t.Fatalf("InitWorkspace() failed: %v", err)
	}

	// Create multiple job structures
	jobIDs := []string{"job_1", "job_2", "job_3"}
	for _, jobID := range jobIDs {
		if err := ws.CreateJobStructure(jobID); err != nil {
			t.Fatalf("CreateJobStructure(%s) failed: %v", jobID, err)
		}
	}

	// Verify all job structures exist
	for _, jobID := range jobIDs {
		jobPath, _ := ws.GetJobPath(jobID)
		if _, err := os.Stat(jobPath); os.IsNotExist(err) {
			t.Errorf("Job directory for %s was not created", jobID)
		}
	}
}

func TestInitWorkspaceFileContent(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := ws.InitWorkspace(); err != nil {
		t.Fatalf("InitWorkspace() failed: %v", err)
	}

	// Verify OKR.md content
	okriPath := filepath.Join(ws.rickDir, OKRFileName)
	content, err := os.ReadFile(okriPath)
	if err != nil {
		t.Fatalf("Failed to read OKR.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("OKR.md is empty")
	}

	// Verify SPEC.md content
	specPath := filepath.Join(ws.rickDir, SpecFileName)
	content, err = os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read SPEC.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("SPEC.md is empty")
	}
}

func TestGetHomeDir(t *testing.T) {
	// GetHomeDir should work in normal conditions
	homeDir, err := GetHomeDir()
	if err != nil {
		t.Fatalf("GetHomeDir() failed: %v", err)
	}
	if homeDir == "" {
		t.Error("GetHomeDir() returned empty string")
	}
}

func TestGetProjectName(t *testing.T) {
	// Save and restore cwd
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Case 1: fallback to directory base name (no go.mod, no PROJECT.md)
	t.Run("fallback to dir name", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to chdir: %v", err)
		}
		name, err := GetProjectName()
		if err != nil {
			t.Fatalf("GetProjectName() error: %v", err)
		}
		if name != filepath.Base(tempDir) {
			t.Errorf("expected %s, got %s", filepath.Base(tempDir), name)
		}
	})

	// Case 2: go.mod module name
	t.Run("go.mod module name", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to chdir: %v", err)
		}
		goModContent := "module github.com/example/myproject\n\ngo 1.21\n"
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatalf("Failed to write go.mod: %v", err)
		}
		name, err := GetProjectName()
		if err != nil {
			t.Fatalf("GetProjectName() error: %v", err)
		}
		if name != "myproject" {
			t.Errorf("expected myproject, got %s", name)
		}
	})

	// Case 3: PROJECT.md first line takes priority
	t.Run("PROJECT.md first line", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to chdir: %v", err)
		}
		goModContent := "module github.com/example/myproject\n\ngo 1.21\n"
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Fatalf("Failed to write go.mod: %v", err)
		}
		rickDir := filepath.Join(tempDir, RickDirName)
		if err := os.MkdirAll(rickDir, 0755); err != nil {
			t.Fatalf("Failed to create .rick dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(rickDir, "PROJECT.md"), []byte("# My Custom Project\nsome description\n"), 0644); err != nil {
			t.Fatalf("Failed to write PROJECT.md: %v", err)
		}
		name, err := GetProjectName()
		if err != nil {
			t.Fatalf("GetProjectName() error: %v", err)
		}
		if name != "My Custom Project" {
			t.Errorf("expected 'My Custom Project', got %s", name)
		}
	})
}

func TestJobPathHierarchy(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	ws, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Test path hierarchy
	jobID := "test_job"
	jobPath, _ := ws.GetJobPath(jobID)
	planDir, _ := GetJobPlanDir(jobID)
	doingDir, _ := GetJobDoingDir(jobID)
	learningDir, _ := GetJobLearningDir(jobID)

	// Verify that subdirectories are under job path
	if !filepath.HasPrefix(planDir, jobPath) {
		t.Error("Plan directory is not under job directory")
	}
	if !filepath.HasPrefix(doingDir, jobPath) {
		t.Error("Doing directory is not under job directory")
	}
	if !filepath.HasPrefix(learningDir, jobPath) {
		t.Error("Learning directory is not under job directory")
	}
}

func TestGetRFCDir(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	rfcDir, err := GetRFCDir()
	if err != nil {
		t.Fatalf("GetRFCDir() failed: %v", err)
	}
	if rfcDir == "" {
		t.Error("GetRFCDir() returned empty string")
	}
	if !strings.Contains(rfcDir, ".rick") {
		t.Errorf("GetRFCDir() = %s, expected to contain '.rick'", rfcDir)
	}
	if !strings.Contains(rfcDir, "RFC") {
		t.Errorf("GetRFCDir() = %s, expected to contain 'RFC'", rfcDir)
	}
}
