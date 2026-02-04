package video

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestGetDuration(t *testing.T) {
	// Skip if ffprobe is not available
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not found, skipping test")
	}

	// Create a test video file
	testVideo := createTestVideo(t)
	defer func() { _ = os.Remove(testVideo) }()

	duration, err := GetDuration(testVideo)
	if err != nil {
		t.Fatalf("GetDuration failed: %v", err)
	}

	// Test video should be approximately 1 second
	if duration < 500*time.Millisecond || duration > 2*time.Second {
		t.Errorf("Expected duration around 1s, got %v", duration)
	}
}

func TestGetDurationNonexistent(t *testing.T) {
	_, err := GetDuration("/nonexistent/video.mp4")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestExtractFrames(t *testing.T) {
	// Skip if ffmpeg is not available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found, skipping test")
	}

	// Create a test video file
	testVideo := createTestVideo(t)
	defer func() { _ = os.Remove(testVideo) }()

	frames, err := ExtractFrames(testVideo, time.Second, nil)
	if err != nil {
		t.Fatalf("ExtractFrames failed: %v", err)
	}
	defer CleanupFrames(frames)

	if len(frames) == 0 {
		t.Error("Expected at least one frame")
	}

	// Verify frames have valid paths
	for _, frame := range frames {
		if _, err := os.Stat(frame.Path); os.IsNotExist(err) {
			t.Errorf("Frame file does not exist: %s", frame.Path)
		}
	}
}

func TestExtractFramesWithProgress(t *testing.T) {
	// Skip if ffmpeg is not available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found, skipping test")
	}

	// Create a test video file
	testVideo := createTestVideo(t)
	defer func() { _ = os.Remove(testVideo) }()

	var progressCalled bool
	frames, err := ExtractFrames(testVideo, time.Second, func(pct float64) {
		progressCalled = true
		if pct < 0 || pct > 1 {
			t.Errorf("Progress out of range: %f", pct)
		}
	})
	if err != nil {
		t.Fatalf("ExtractFrames failed: %v", err)
	}
	defer CleanupFrames(frames)

	if !progressCalled {
		t.Log("Progress callback was not called (may be due to short video)")
	}
}

func TestExtractFramesNonexistent(t *testing.T) {
	_, err := ExtractFrames("/nonexistent/video.mp4", 0, nil)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestCleanupFrames(t *testing.T) {
	// Create temp directory with a fake frame
	tempDir, err := os.MkdirTemp("", "memorex-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	framePath := filepath.Join(tempDir, "0001.png")
	if err := os.WriteFile(framePath, []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	frames := []Frame{{Path: framePath}}
	CleanupFrames(frames)

	// Verify directory was removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("Expected temp directory to be removed")
		_ = os.RemoveAll(tempDir)
	}
}

func TestCleanupFramesEmpty(t *testing.T) {
	// Should not panic with empty slice
	CleanupFrames([]Frame{})
	CleanupFrames(nil)
}

// createTestVideo creates a minimal test video using ffmpeg
func createTestVideo(t *testing.T) string {
	t.Helper()

	tempFile, err := os.CreateTemp("", "memorex-test-*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Create a 1-second test video with solid color
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "color=c=red:size=320x240:rate=30:duration=1",
		"-c:v", "libx264",
		"-t", "1",
		"-y",
		tempFile.Name(),
	)

	if err := cmd.Run(); err != nil {
		_ = os.Remove(tempFile.Name())
		t.Fatalf("Failed to create test video: %v", err)
	}

	return tempFile.Name()
}
