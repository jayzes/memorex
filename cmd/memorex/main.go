// Package main provides the memorex CLI tool for converting video/audio files
// into Claude-friendly markdown with transcripts and keyframes.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jayzes/memorex/internal/audio"
	"github.com/jayzes/memorex/internal/output"
	"github.com/jayzes/memorex/internal/ui"
	"github.com/jayzes/memorex/internal/video"
)

var (
	outputPath   string
	threshold    float64
	quality      int
	scale        float64
	modelPath    string
	noTranscript bool
	noFrames     bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "memorex [options] <video-file>",
		Short: "Convert video/audio files into Claude-friendly markdown",
		Long: `Memorex processes video and audio files to extract transcripts and keyframes,
generating structured markdown suitable for analysis by Claude or other LLMs.`,
		Args: cobra.ExactArgs(1),
		RunE: run,
	}

	homeDir, _ := os.UserHomeDir()
	defaultModel := filepath.Join(homeDir, ".cache", "whisper", "ggml-base.bin")

	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: <input>_memorex.md)")
	rootCmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.85, "Frame similarity threshold 0.0-1.0")
	rootCmd.Flags().IntVarP(&quality, "quality", "q", 30, "JPEG quality 1-100")
	rootCmd.Flags().Float64VarP(&scale, "scale", "s", 0.5, "Frame scale factor")
	rootCmd.Flags().StringVarP(&modelPath, "model", "m", defaultModel, "Whisper model path")
	rootCmd.Flags().BoolVar(&noTranscript, "no-transcript", false, "Skip audio transcription")
	rootCmd.Flags().BoolVar(&noFrames, "no-frames", false, "Skip frame extraction (audio only)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(_ *cobra.Command, args []string) error {
	inputPath := args[0]

	// Validate input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputPath)
	}

	// Determine output path
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		outputPath = base + "_memorex.md"
	}

	// Create frames directory
	framesDir := strings.TrimSuffix(outputPath, ".md") + "_frames"
	if !noFrames {
		if err := os.MkdirAll(framesDir, 0o750); err != nil {
			return fmt.Errorf("failed to create frames directory: %w", err)
		}
	}

	ui.PrintHeader("memorex")
	ui.PrintInfo(fmt.Sprintf("Processing: %s", filepath.Base(inputPath)))

	// Get video duration
	duration, err := video.GetDuration(inputPath)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not get duration: %v", err))
	} else {
		ui.PrintInfo(fmt.Sprintf("Duration: %s", formatDuration(duration)))
	}
	fmt.Fprintln(os.Stderr)

	var keyframes []video.Keyframe
	var totalFrames int

	// Extract and process frames
	if !noFrames {
		// Step 1: Extract frames
		step := ui.NewStep("Extracting frames")
		frames, err := video.ExtractFrames(inputPath, duration, step.Update)
		if err != nil {
			step.Error("Frame extraction failed")
			return fmt.Errorf("frame extraction failed: %w", err)
		}
		totalFrames = len(frames)
		step.Complete(fmt.Sprintf("Extracted %d frames", totalFrames))

		// Step 2: Detect keyframes
		step = ui.NewStep("Detecting keyframes")
		keyframes, err = video.DetectKeyframes(frames, threshold, step.Update)
		if err != nil {
			step.Error("Keyframe detection failed")
			return fmt.Errorf("keyframe detection failed: %w", err)
		}
		step.Complete(fmt.Sprintf("Found %d keyframes", len(keyframes)))

		// Step 3: Save keyframes
		step = ui.NewStep("Saving keyframes")
		if err := video.SaveKeyframes(keyframes, framesDir, quality, scale, step.Update); err != nil {
			step.Error("Failed to save keyframes")
			return fmt.Errorf("failed to save keyframes: %w", err)
		}
		step.Complete("Keyframes saved")
	}

	var segments []audio.Segment

	// Transcribe audio
	if !noTranscript {
		// Step: Download model if needed
		if !audio.ModelExists(modelPath) {
			step := ui.NewStep("Downloading whisper model")
			if err := audio.DownloadModel(modelPath, step.Update); err != nil {
				step.Error("Model download failed")
				return fmt.Errorf("failed to download model: %w", err)
			}
			step.Complete("Model downloaded")
		}

		// Step: Extract audio
		step := ui.NewStep("Extracting audio")
		audioPath, err := audio.ExtractAudioTrack(inputPath, duration, step.Update)
		if err != nil {
			step.Error("Audio extraction failed")
			return fmt.Errorf("audio extraction failed: %w", err)
		}
		step.Complete("Audio extracted")

		// Step: Transcribe
		step = ui.NewStep("Transcribing")
		segments, err = audio.TranscribeAudio(audioPath, modelPath, step.Update)
		// Clean up audio file
		_ = os.Remove(audioPath)
		if err != nil {
			step.Error("Transcription failed")
			return fmt.Errorf("transcription failed: %w", err)
		}
		step.Complete(fmt.Sprintf("Transcribed %d segments", len(segments)))
	}

	// Step: Generate markdown
	step := ui.NewStep("Generating markdown")
	result := output.Result{
		InputPath:   inputPath,
		Duration:    duration,
		TotalFrames: totalFrames,
		Keyframes:   convertKeyframes(keyframes, framesDir),
		Segments:    convertSegments(segments),
	}

	if err := output.WriteMarkdown(outputPath, result); err != nil {
		step.Error("Failed to write output")
		return fmt.Errorf("failed to write output: %w", err)
	}
	step.Complete("Markdown generated")

	// Print summary
	fmt.Fprintln(os.Stderr)
	ui.PrintSuccess(fmt.Sprintf("Output: %s", outputPath))
	if !noFrames {
		ui.PrintInfo(fmt.Sprintf("Frames: %s/", framesDir))
	}

	tokenEstimate := output.EstimateTokens(result)
	ui.PrintInfo(fmt.Sprintf("Estimated tokens: ~%d", tokenEstimate))

	return nil
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func convertKeyframes(keyframes []video.Keyframe, framesDir string) []output.Keyframe {
	result := make([]output.Keyframe, len(keyframes))
	for i, kf := range keyframes {
		result[i] = output.Keyframe{
			Index:     kf.Index,
			Timestamp: kf.Timestamp,
			Path:      filepath.Join(framesDir, fmt.Sprintf("frame_%04d.jpg", kf.Index)),
		}
	}
	return result
}

func convertSegments(segments []audio.Segment) []output.Segment {
	result := make([]output.Segment, len(segments))
	for i, seg := range segments {
		result[i] = output.Segment{
			Start: seg.Start,
			End:   seg.End,
			Text:  seg.Text,
		}
	}
	return result
}
