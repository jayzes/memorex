package video

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectKeyframesEmpty(t *testing.T) {
	keyframes, err := DetectKeyframes(nil, 0.85, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(keyframes) != 0 {
		t.Errorf("Expected empty keyframes, got %d", len(keyframes))
	}
}

func TestDetectKeyframesSingleFrame(t *testing.T) {
	tempDir := t.TempDir()
	framePath := createTestImage(t, tempDir, "0001.png", color.RGBA{255, 0, 0, 255})

	frames := []Frame{{
		Path:      framePath,
		Index:     1,
		Timestamp: 0,
	}}

	keyframes, err := DetectKeyframes(frames, 0.85, nil)
	if err != nil {
		t.Fatalf("DetectKeyframes failed: %v", err)
	}

	if len(keyframes) != 1 {
		t.Errorf("Expected 1 keyframe, got %d", len(keyframes))
	}
}

func TestDetectKeyframesIdenticalFrames(t *testing.T) {
	tempDir := t.TempDir()

	// Create 3 identical frames
	var frames []Frame
	for i := 1; i <= 3; i++ {
		framePath := createTestImage(t, tempDir, "%04d.png", color.RGBA{255, 0, 0, 255})
		frames = append(frames, Frame{
			Path:      framePath,
			Index:     i,
			Timestamp: time.Duration(i-1) * time.Second,
		})
	}

	keyframes, err := DetectKeyframes(frames, 0.85, nil)
	if err != nil {
		t.Fatalf("DetectKeyframes failed: %v", err)
	}

	// Should include first and last only (since they're identical)
	if len(keyframes) != 2 {
		t.Errorf("Expected 2 keyframes (first and last), got %d", len(keyframes))
	}
}

func TestDetectKeyframesDifferentFrames(t *testing.T) {
	tempDir := t.TempDir()

	// Create frames with different colors
	// Note: Solid color frames have zero variance, so NCC returns 1.0 (identical)
	// This test verifies first and last frames are always included
	colors := []color.RGBA{
		{255, 0, 0, 255}, // Red
		{0, 255, 0, 255}, // Green
		{0, 0, 255, 255}, // Blue
	}

	frames := make([]Frame, 0, len(colors))
	for i, c := range colors {
		idx := i + 1
		framePath := createTestImage(t, tempDir, "%04d.png", c)
		frames = append(frames, Frame{
			Path:      framePath,
			Index:     idx,
			Timestamp: time.Duration(i) * time.Second,
		})
	}

	keyframes, err := DetectKeyframes(frames, 0.85, nil)
	if err != nil {
		t.Fatalf("DetectKeyframes failed: %v", err)
	}

	// First and last frames should always be included
	if len(keyframes) < 2 {
		t.Errorf("Expected at least 2 keyframes (first and last), got %d", len(keyframes))
	}

	// Verify first frame is included
	if keyframes[0].Index != 1 {
		t.Errorf("Expected first keyframe to have index 1, got %d", keyframes[0].Index)
	}

	// Verify last frame is included
	if keyframes[len(keyframes)-1].Index != 3 {
		t.Errorf("Expected last keyframe to have index 3, got %d", keyframes[len(keyframes)-1].Index)
	}
}

func TestNormalizedCrossCorrelation(t *testing.T) {
	// Test identical arrays
	a := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	ncc := normalizedCrossCorrelation(a, a)
	if ncc < 0.99 {
		t.Errorf("Expected NCC ~1.0 for identical arrays, got %f", ncc)
	}

	// Test different arrays
	b := []float64{0.5, 0.4, 0.3, 0.2, 0.1}
	ncc = normalizedCrossCorrelation(a, b)
	if ncc > 0 {
		t.Errorf("Expected negative NCC for inversely correlated arrays, got %f", ncc)
	}

	// Test empty arrays
	ncc = normalizedCrossCorrelation([]float64{}, []float64{})
	if ncc != 0 {
		t.Errorf("Expected 0 for empty arrays, got %f", ncc)
	}

	// Test different length arrays
	ncc = normalizedCrossCorrelation(a, []float64{0.1, 0.2})
	if ncc != 0 {
		t.Errorf("Expected 0 for different length arrays, got %f", ncc)
	}

	// Test constant arrays
	constant := []float64{0.5, 0.5, 0.5, 0.5, 0.5}
	ncc = normalizedCrossCorrelation(constant, constant)
	if ncc != 1.0 {
		t.Errorf("Expected 1.0 for constant arrays, got %f", ncc)
	}
}

func TestSaveKeyframes(t *testing.T) {
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// Create a test frame
	framePath := createTestImage(t, tempDir, "0001.png", color.RGBA{255, 0, 0, 255})
	keyframes := []Keyframe{{
		Path:      framePath,
		Index:     1,
		Timestamp: 0,
	}}

	err := SaveKeyframes(keyframes, outputDir, 30, 0.5, nil)
	if err != nil {
		t.Fatalf("SaveKeyframes failed: %v", err)
	}

	// Verify output file exists
	outputPath := filepath.Join(outputDir, "frame_0001.jpg")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}
}

func TestSaveKeyframesInvalidPath(t *testing.T) {
	keyframes := []Keyframe{{
		Path:      "/nonexistent/frame.png",
		Index:     1,
		Timestamp: 0,
	}}

	err := SaveKeyframes(keyframes, "/tmp", 30, 0.5, nil)
	if err == nil {
		t.Error("Expected error for nonexistent frame path")
	}
}

// createTestImage creates a solid color PNG image for testing
func createTestImage(t *testing.T, dir, _ string, c color.Color) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, c)
		}
	}

	file, err := os.CreateTemp(dir, "frame_*.png")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = file.Close() }()

	if err := png.Encode(file, img); err != nil {
		t.Fatalf("Failed to encode PNG: %v", err)
	}

	return file.Name()
}
