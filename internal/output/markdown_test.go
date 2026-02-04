package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, "0:00"},
		{30 * time.Second, "0:30"},
		{time.Minute, "1:00"},
		{90 * time.Second, "1:30"},
		{time.Hour, "1:00:00"},
		{time.Hour + 30*time.Minute + 45*time.Second, "1:30:45"},
		{2*time.Hour + 5*time.Minute + 3*time.Second, "2:05:03"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.input)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	// Empty result
	empty := Result{}
	tokens := EstimateTokens(empty)
	if tokens < 100 {
		t.Errorf("Expected at least 100 tokens for metadata, got %d", tokens)
	}

	// Result with segments
	withSegments := Result{
		Segments: []Segment{
			{Text: "Hello world this is a test"},
			{Text: "More words here for testing purposes"},
		},
	}
	tokensWithSeg := EstimateTokens(withSegments)
	if tokensWithSeg <= tokens {
		t.Error("Expected more tokens with segments")
	}

	// Result with keyframes
	withKeyframes := Result{
		Keyframes: []Keyframe{
			{Index: 1},
			{Index: 2},
			{Index: 3},
		},
	}
	tokensWithKf := EstimateTokens(withKeyframes)
	// 3 keyframes * ~1000 tokens each + metadata
	if tokensWithKf < 3000 {
		t.Errorf("Expected at least 3000 tokens for 3 keyframes, got %d", tokensWithKf)
	}
}

func TestWriteMarkdown(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test_memorex.md")
	framesDir := filepath.Join(tempDir, "test_memorex_frames")
	if err := os.MkdirAll(framesDir, 0o750); err != nil {
		t.Fatalf("Failed to create frames dir: %v", err)
	}

	result := Result{
		InputPath:   "/path/to/video.mp4",
		Duration:    2*time.Minute + 34*time.Second,
		TotalFrames: 154,
		Keyframes: []Keyframe{
			{Index: 1, Timestamp: 0, Path: filepath.Join(framesDir, "frame_0001.jpg")},
			{Index: 15, Timestamp: 15 * time.Second, Path: filepath.Join(framesDir, "frame_0015.jpg")},
		},
		Segments: []Segment{
			{Start: 0, End: 5 * time.Second, Text: "Hello world"},
			{Start: 5 * time.Second, End: 10 * time.Second, Text: "This is a test"},
		},
	}

	err := WriteMarkdown(outputPath, result)
	if err != nil {
		t.Fatalf("WriteMarkdown failed: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	checks := []string{
		"# Video Analysis: video.mp4",
		"Duration: 2:34",
		"Original frames: 154",
		"Keyframes extracted: 2",
		"## Transcript",
		"[0:00] Hello world",
		"[0:05] This is a test",
		"## Keyframes",
		"### Frame 1 (0:00)",
		"### Frame 15 (0:15)",
	}

	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Output missing expected content: %s", check)
		}
	}
}

func TestWriteMarkdownNoSegments(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test.md")

	result := Result{
		InputPath:   "/path/to/video.mp4",
		Duration:    time.Minute,
		TotalFrames: 60,
		Keyframes: []Keyframe{
			{Index: 1, Timestamp: 0, Path: "/tmp/frame.jpg"},
		},
	}

	err := WriteMarkdown(outputPath, result)
	if err != nil {
		t.Fatalf("WriteMarkdown failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	// Should not have transcript section (or it should be empty)
	if strings.Contains(string(content), "[0:") && strings.Contains(string(content), "## Transcript") {
		// Check if there's content after transcript header
		lines := strings.Split(string(content), "\n")
		inTranscript := false
		hasTranscriptContent := false
		for _, line := range lines {
			if strings.Contains(line, "## Transcript") {
				inTranscript = true
				continue
			}
			if inTranscript && strings.HasPrefix(line, "##") {
				break
			}
			if inTranscript && strings.HasPrefix(line, "[") {
				hasTranscriptContent = true
			}
		}
		if hasTranscriptContent {
			t.Error("Expected no transcript content")
		}
	}
}

func TestWriteMarkdownNoKeyframes(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test.md")

	result := Result{
		InputPath:   "/path/to/audio.mp3",
		Duration:    time.Minute,
		TotalFrames: 0,
		Segments: []Segment{
			{Start: 0, End: 5 * time.Second, Text: "Audio content"},
		},
	}

	err := WriteMarkdown(outputPath, result)
	if err != nil {
		t.Fatalf("WriteMarkdown failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	// Should have transcript but no keyframes section content
	if !strings.Contains(string(content), "Audio content") {
		t.Error("Expected transcript content")
	}
}

func TestWriteMarkdownInvalidPath(t *testing.T) {
	result := Result{InputPath: "test.mp4"}
	err := WriteMarkdown("/nonexistent/directory/test.md", result)
	if err == nil {
		t.Error("Expected error for invalid output path")
	}
}
