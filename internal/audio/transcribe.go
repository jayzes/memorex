// Package audio provides audio transcription using whisper-cli.
package audio

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultModelURL is the URL to download the ggml-base.en model
	DefaultModelURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin"
	// ModelSize is the approximate size of the model for progress calculation
	ModelSize = 148_000_000 // ~148MB
)

// Segment represents a transcribed audio segment with timing
type Segment struct {
	Start time.Duration
	End   time.Duration
	Text  string
}

// ProgressFunc is called with progress updates (0.0 to 1.0)
type ProgressFunc func(percent float64)

// ModelExists checks if the whisper model exists at the given path.
func ModelExists(modelPath string) bool {
	_, err := os.Stat(modelPath)
	return err == nil
}

// DownloadModel downloads the whisper model to the specified path.
func DownloadModel(modelPath string, onProgress ProgressFunc) error {
	return downloadModel(modelPath, onProgress)
}

// ExtractAudioTrack extracts audio from a video file with progress reporting.
func ExtractAudioTrack(inputPath string, duration time.Duration, onProgress ProgressFunc) (string, error) {
	return extractAudio(inputPath, duration, onProgress)
}

// TranscribeAudio transcribes an audio file using whisper.
func TranscribeAudio(audioPath, modelPath string, onProgress ProgressFunc) ([]Segment, error) {
	return runWhisper(audioPath, modelPath, onProgress)
}

// Transcribe extracts audio from video and transcribes it using whisper-cli.
// This is a convenience function that combines ExtractAudioTrack and TranscribeAudio.
func Transcribe(inputPath, modelPath string, duration time.Duration, onProgress ProgressFunc) ([]Segment, error) {
	// Check if model exists
	if !ModelExists(modelPath) {
		return nil, fmt.Errorf("whisper model not found at %s", modelPath)
	}

	// Extract audio (0-50%)
	var extractProgress ProgressFunc
	if onProgress != nil {
		extractProgress = func(p float64) {
			onProgress(p * 0.5)
		}
	}

	audioPath, err := extractAudio(inputPath, duration, extractProgress)
	if err != nil {
		return nil, fmt.Errorf("audio extraction failed: %w", err)
	}
	defer func() { _ = os.Remove(audioPath) }()

	// Transcribe (50-100%)
	var whisperProgress ProgressFunc
	if onProgress != nil {
		whisperProgress = func(p float64) {
			onProgress(0.5 + p*0.5)
		}
	}

	segments, err := runWhisper(audioPath, modelPath, whisperProgress)
	if err != nil {
		return nil, fmt.Errorf("whisper transcription failed: %w", err)
	}

	return segments, nil
}

// downloadModel downloads the whisper model to the specified path
func downloadModel(modelPath string, onProgress ProgressFunc) error {
	// Create the directory if it doesn't exist
	modelDir := filepath.Dir(modelPath)
	if err := os.MkdirAll(modelDir, 0o750); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Download the model
	resp, err := http.Get(DefaultModelURL)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download model: HTTP %d", resp.StatusCode)
	}

	// Get content length for progress
	contentLength := resp.ContentLength
	if contentLength <= 0 {
		contentLength = ModelSize // Use approximate size as fallback
	}

	// Create temp file for download
	tempFile, err := os.CreateTemp(modelDir, "whisper-model-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	// Copy with progress tracking
	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := tempFile.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write model: %w", writeErr)
			}
			written += int64(n)
			if onProgress != nil {
				pct := float64(written) / float64(contentLength)
				if pct > 1.0 {
					pct = 1.0
				}
				onProgress(pct)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read model: %w", readErr)
		}
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Move temp file to final location
	if err := os.Rename(tempPath, modelPath); err != nil {
		return fmt.Errorf("failed to move model to final location: %w", err)
	}

	return nil
}

// extractAudio extracts audio from video to a WAV file suitable for Whisper
func extractAudio(inputPath string, duration time.Duration, onProgress ProgressFunc) (string, error) {
	// Create temp file for audio
	tempFile, err := os.CreateTemp("", "memorex-audio-*.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	audioPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Extract audio using FFmpeg
	// - 16kHz sample rate (required by Whisper)
	// - Mono channel
	// - 16-bit PCM WAV format
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		"-y",
		"-loglevel", "error",
		"-progress", "pipe:1",
		"-nostats",
		audioPath,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = os.Remove(audioPath)
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = os.Remove(audioPath)
		return "", fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Parse progress output
	if onProgress != nil && duration > 0 {
		go parseFFmpegProgress(stdout, duration, onProgress)
	}

	if err := cmd.Wait(); err != nil {
		_ = os.Remove(audioPath)
		return "", fmt.Errorf("ffmpeg audio extraction failed: %w", err)
	}

	return audioPath, nil
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

// runWhisper runs the whisper-cli command and parses the output
func runWhisper(audioPath, modelPath string, onProgress ProgressFunc) ([]Segment, error) {
	// Create temp file for output
	outputFile, err := os.CreateTemp("", "memorex-transcript-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	outputPath := outputFile.Name()
	if err := outputFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}
	defer func() { _ = os.Remove(outputPath) }()

	// Try whisper-cli first, then fall back to whisper
	whisperCmd := "whisper-cli"
	if _, err := exec.LookPath(whisperCmd); err != nil {
		whisperCmd = "whisper"
		if _, err := exec.LookPath(whisperCmd); err != nil {
			// Try the path where make install-whisper puts it
			whisperCmd = os.ExpandEnv("$HOME/.local/share/whisper.cpp/src/build/bin/whisper-cli")
			if _, err := os.Stat(whisperCmd); err != nil {
				return nil, fmt.Errorf("whisper-cli not found. Install whisper.cpp and ensure whisper-cli is in PATH")
			}
		}
	}

	// Run whisper with timestamps
	cmd := exec.Command(whisperCmd,
		"-m", modelPath,
		"-f", audioPath,
		"-otxt",
		"-of", strings.TrimSuffix(outputPath, ".txt"),
		"--print-progress", // Enable progress output
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start whisper: %w", err)
	}

	// Parse progress and collect output
	var outputBuilder strings.Builder
	done := make(chan struct{})

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line)
			outputBuilder.WriteString("\n")
		}
		close(done)
	}()

	// Parse whisper progress from stderr (format: "whisper_print_progress_callback: progress = XX%")
	if onProgress != nil {
		go parseWhisperProgress(stderr, onProgress)
	}

	<-done

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("whisper failed: %w", err)
	}

	output := outputBuilder.String()

	// Parse the whisper output from stderr/stdout which contains timestamps
	segments := parseWhisperOutput(output)
	if len(segments) == 0 {
		// Fall back to reading the output file without timestamps
		content, err := os.ReadFile(outputPath)
		if err == nil && len(content) > 0 {
			segments = []Segment{{
				Start: 0,
				End:   0,
				Text:  strings.TrimSpace(string(content)),
			}}
		}
	}

	return segments, nil
}

// parseWhisperProgress parses whisper-cli progress output
func parseWhisperProgress(stderr io.Reader, onProgress ProgressFunc) {
	scanner := bufio.NewScanner(stderr)
	progressPattern := regexp.MustCompile(`progress\s*=\s*(\d+)%`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := progressPattern.FindStringSubmatch(line)
		if matches != nil {
			pct, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			onProgress(float64(pct) / 100.0)
		}
	}
}

// parseWhisperOutput parses whisper-cli output with timestamps
// Format: [00:00:00.000 --> 00:00:05.000] Text here
func parseWhisperOutput(output string) []Segment {
	var segments []Segment

	// Pattern matches: [HH:MM:SS.mmm --> HH:MM:SS.mmm] text
	pattern := regexp.MustCompile(`\[(\d{2}:\d{2}:\d{2}\.\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2}\.\d{3})\]\s*(.*)`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		matches := pattern.FindStringSubmatch(line)
		if matches != nil {
			start, _ := parseTimestamp(matches[1])
			end, _ := parseTimestamp(matches[2])
			text := strings.TrimSpace(matches[3])

			if text != "" {
				segments = append(segments, Segment{
					Start: start,
					End:   end,
					Text:  text,
				})
			}
		}
	}

	return segments
}

// parseTimestamp parses HH:MM:SS.mmm format
func parseTimestamp(ts string) (time.Duration, error) {
	parts := strings.Split(ts, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid timestamp format: %s", ts)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	secParts := strings.Split(parts[2], ".")
	seconds, err := strconv.Atoi(secParts[0])
	if err != nil {
		return 0, err
	}

	var millis int
	if len(secParts) > 1 {
		millis, err = strconv.Atoi(secParts[1])
		if err != nil {
			return 0, err
		}
	}

	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(millis)*time.Millisecond

	return duration, nil
}
