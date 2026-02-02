package jobs

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
)

func TestProcessor_AppUpdate(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	appsDir := filepath.Join(tmpDir, "apps")
	os.MkdirAll(appsDir, 0755)

	// Create test database
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create test app
	app := db.NewApp("test-app", "Test app", "version: '3'\nservices:\n  web:\n    image: nginx:latest\n")
	app.Status = constants.AppStatusRunning
	app.NodeID = "test-node"
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Create app directory with compose file
	dockerMgr := docker.NewManager(appsDir)
	if err := dockerMgr.CreateAppDirectory(app.Name, app.ComposeContent); err != nil {
		t.Fatalf("Failed to create app directory: %v", err)
	}

	// Create mock docker manager that simulates successful update
	mockExecutor := docker.NewMockCommandExecutor()
	mockExecutor.SetMockOutput("docker", []string{"compose", "pull", "--ignore-buildable"}, []byte("Pulling..."))
	mockExecutor.SetMockOutput("docker", []string{"compose", "up", "-d", "--build"}, []byte("Creating..."))
	dockerMgrWithMock := docker.NewManagerWithExecutor(appsDir, mockExecutor)

	// Create job
	job := db.NewJob(constants.JobTypeAppUpdate, app.ID, nil)
	if err := database.CreateJob(job); err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Create processor with mock services
	// For this test, we need to create minimal mocks for appService and tunnelService
	// Since these aren't used by AppUpdateHandler, we can pass nil
	processor := NewProcessor(
		database,
		dockerMgrWithMock,
		nil, // appService not needed for app_update
		nil, // tunnelService not needed for app_update
		slog.Default(),
	)

	// Process the job
	ctx := context.Background()
	if err := processor.ProcessJob(ctx, job); err != nil {
		t.Fatalf("Job processing failed: %v", err)
	}

	// Verify job was marked as completed
	updatedJob, err := database.GetJob(job.ID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	if updatedJob.Status != constants.JobStatusCompleted {
		t.Errorf("Expected job status to be 'completed', got '%s'", updatedJob.Status)
	}

	if updatedJob.Progress != 100 {
		t.Errorf("Expected job progress to be 100, got %d", updatedJob.Progress)
	}

	// Verify app status was updated to running
	updatedApp, err := database.GetApp(app.ID)
	if err != nil {
		t.Fatalf("Failed to get updated app: %v", err)
	}

	if updatedApp.Status != constants.AppStatusRunning {
		t.Errorf("Expected app status to be 'running', got '%s'", updatedApp.Status)
	}
}

func TestWorker_ConcurrencyControl(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create test app
	app := db.NewApp("test-app", "Test app", "version: '3'\nservices:\n  web:\n    image: nginx:latest\n")
	app.NodeID = "test-node"
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Create two jobs for the same app
	job1 := db.NewJob(constants.JobTypeAppUpdate, app.ID, nil)
	job2 := db.NewJob(constants.JobTypeAppUpdate, app.ID, nil)

	if err := database.CreateJob(job1); err != nil {
		t.Fatalf("Failed to create job 1: %v", err)
	}

	if err := database.CreateJob(job2); err != nil {
		t.Fatalf("Failed to create job 2: %v", err)
	}

	// Verify that GetActiveJobForApp returns an active job (could be either job1 or job2)
	activeJob, err := database.GetActiveJobForApp(app.ID)
	if err != nil {
		t.Fatalf("Failed to get active job: %v", err)
	}

	if activeJob == nil {
		t.Fatal("Expected to find an active job, got nil")
	}

	// The active job should be one of the two jobs we created
	if activeJob.ID != job1.ID && activeJob.ID != job2.ID {
		t.Errorf("Expected active job to be job1 or job2, got %s", activeJob.ID)
	}

	// Verify the job is in pending state
	if activeJob.Status != constants.JobStatusPending {
		t.Errorf("Expected active job status to be 'pending', got '%s'", activeJob.Status)
	}
}

func TestDB_MarkStaleJobs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Create test app
	app := db.NewApp("test-app", "Test app", "version: '3'\nservices:\n  web:\n    image: nginx:latest\n")
	app.NodeID = "test-node"
	if err := database.CreateApp(app); err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Create a job and mark it as running
	job := db.NewJob(constants.JobTypeAppUpdate, app.ID, nil)
	if err := database.CreateJob(job); err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Mark job as running
	msg := "Processing..."
	if err := database.UpdateJobStatus(job.ID, constants.JobStatusRunning, 50, &msg); err != nil {
		t.Fatalf("Failed to update job status: %v", err)
	}

	// Simulate stale job by updating the updated_at timestamp to be old
	// (In reality, we'd wait 30 minutes, but we'll manipulate the database for testing)
	oldTime := time.Now().Add(-40 * time.Minute)
	if _, err := database.Exec("UPDATE jobs SET updated_at = ? WHERE id = ?", oldTime, job.ID); err != nil {
		t.Fatalf("Failed to set old timestamp: %v", err)
	}

	// Mark stale jobs as failed
	if err := database.MarkStaleJobsAsFailed(30 * time.Minute); err != nil {
		t.Fatalf("Failed to mark stale jobs: %v", err)
	}

	// Verify job was marked as failed
	updatedJob, err := database.GetJob(job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if updatedJob.Status != constants.JobStatusFailed {
		t.Errorf("Expected job status to be 'failed', got '%s'", updatedJob.Status)
	}

	if updatedJob.ErrorMessage == nil || *updatedJob.ErrorMessage == "" {
		t.Error("Expected error message to be set")
	}
}
