package audio

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestExtractAudio(t *testing.T) {
	// Skip if ffmpeg is not available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found, skipping test")
	}

	// Create a test video with audio
	testVideo := createTestVideoWithAudio(t)
	defer func() { _ = os.Remove(testVideo) }()

	audioPath, err := extractAudio(testVideo, time.Second, nil)
	if err != nil {
		t.Fatalf("extractAudio failed: %v", err)
	}
	defer func() { _ = os.Remove(audioPath) }()

	// Verify the audio file exists and has content
	info, err := os.Stat(audioPath)
	if err != nil {
		t.Fatalf("Audio file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Audio file is empty")
	}
}

func TestExtractAudioNonexistent(t *testing.T) {
	_, err := extractAudio("/nonexistent/video.mp4", 0, nil)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"00:00:00.000", 0, false},
		{"00:00:01.000", time.Second, false},
		{"00:01:00.000", time.Minute, false},
		{"01:00:00.000", time.Hour, false},
		{"00:00:00.500", 500 * time.Millisecond, false},
		{"01:23:45.678", time.Hour + 23*time.Minute + 45*time.Second + 678*time.Millisecond, false},
		{"invalid", 0, true},
		{"00:00", 0, true},
		{"00:00:00", 0, false}, // No milliseconds
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseTimestamp(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %s", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("For input %s, expected %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}

func TestParseWhisperOutput(t *testing.T) {
	output := `whisper_init_from_file: loading model from 'model.bin'
[00:00:00.000 --> 00:00:03.000]  Hello, world.
[00:00:03.000 --> 00:00:06.500]  This is a test.
[00:00:06.500 --> 00:00:10.000]  End of transcription.
`

	segments := parseWhisperOutput(output)

	if len(segments) != 3 {
		t.Fatalf("Expected 3 segments, got %d", len(segments))
	}

	// Check first segment
	if segments[0].Text != "Hello, world." {
		t.Errorf("Expected 'Hello, world.', got '%s'", segments[0].Text)
	}
	if segments[0].Start != 0 {
		t.Errorf("Expected start 0, got %v", segments[0].Start)
	}
	if segments[0].End != 3*time.Second {
		t.Errorf("Expected end 3s, got %v", segments[0].End)
	}

	// Check last segment
	if segments[2].Text != "End of transcription." {
		t.Errorf("Expected 'End of transcription.', got '%s'", segments[2].Text)
	}
}

func TestParseWhisperOutputEmpty(t *testing.T) {
	segments := parseWhisperOutput("")
	if len(segments) != 0 {
		t.Errorf("Expected 0 segments for empty output, got %d", len(segments))
	}
}

func TestParseWhisperOutputNoTimestamps(t *testing.T) {
	output := `whisper_init_from_file: loading model
Some random log output
No timestamps here`

	segments := parseWhisperOutput(output)
	if len(segments) != 0 {
		t.Errorf("Expected 0 segments for output without timestamps, got %d", len(segments))
	}
}

func TestTranscribeModelNotFound(t *testing.T) {
	_, err := Transcribe("/some/video.mp4", "/nonexistent/model.bin", 0, nil)
	if err == nil {
		t.Error("Expected error for nonexistent model")
	}
}

// createTestVideoWithAudio creates a minimal test video with audio
func createTestVideoWithAudio(t *testing.T) string {
	t.Helper()

	tempFile, err := os.CreateTemp("", "memorex-test-audio-*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Create a 1-second test video with tone audio
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "color=c=red:size=320x240:rate=30:duration=1",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:v", "libx264",
		"-c:a", "aac",
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
