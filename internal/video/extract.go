// Package video provides video frame extraction and keyframe detection.
package video

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Frame represents an extracted video frame
type Frame struct {
	Path      string
	Index     int
	Timestamp time.Duration
}

// Keyframe represents a frame that differs significantly from its predecessor
type Keyframe struct {
	Path      string
	Index     int
	Timestamp time.Duration
}

// ProgressFunc is called with progress updates (0.0 to 1.0)
type ProgressFunc func(percent float64)

// GetDuration returns the duration of a video file using ffprobe
func GetDuration(inputPath string) (time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return time.Duration(seconds * float64(time.Second)), nil
}

// ExtractFrames extracts frames from a video file at 1 fps
func ExtractFrames(inputPath string, duration time.Duration, onProgress ProgressFunc) ([]Frame, error) {
	// Create temp directory for frames
	tempDir, err := os.MkdirTemp("", "memorex-frames-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Extract frames at 1 fps using FFmpeg
	outputPattern := filepath.Join(tempDir, "%04d.png")
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", "fps=1",
		"-q:v", "2",
		"-loglevel", "error",
		"-progress", "pipe:1", // Output progress to stdout
		"-nostats",
		outputPattern,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Parse progress output
	if onProgress != nil && duration > 0 {
		go parseFFmpegProgress(stdout, duration, onProgress)
	}

	if err := cmd.Wait(); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("ffmpeg extraction failed: %w", err)
	}

	// Read extracted frames
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to read temp directory: %w", err)
	}

	// Parse frame files and create Frame objects
	framePattern := regexp.MustCompile(`^(\d+)\.png$`)
	var frames []Frame

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := framePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		index, _ := strconv.Atoi(matches[1])
		frames = append(frames, Frame{
			Path:      filepath.Join(tempDir, entry.Name()),
			Index:     index,
			Timestamp: time.Duration(index-1) * time.Second, // 1 fps means 1 frame per second
		})
	}

	// Sort frames by index
	sort.Slice(frames, func(i, j int) bool {
		return frames[i].Index < frames[j].Index
	})

	if len(frames) == 0 {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("no frames extracted from video")
	}

	return frames, nil
}

// parseFFmpegProgress reads ffmpeg progress output and calls the callback
func parseFFmpegProgress(stdout io.Reader, totalDuration time.Duration, onProgress ProgressFunc) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_us=") {
			usStr := strings.TrimPrefix(line, "out_time_us=")
			us, err := strconv.ParseInt(usStr, 10, 64)
			if err != nil {
				continue
			}
			currentTime := time.Duration(us) * time.Microsecond
			percent := float64(currentTime) / float64(totalDuration)
			if percent > 1.0 {
				percent = 1.0
			}
			onProgress(percent)
		}
	}
}

// CleanupFrames removes the temporary frame directory
func CleanupFrames(frames []Frame) {
	if len(frames) == 0 {
		return
	}
	tempDir := filepath.Dir(frames[0].Path)
	_ = os.RemoveAll(tempDir)
}
