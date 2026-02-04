// Package output generates markdown files from video analysis results.
package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Keyframe represents a keyframe for output
type Keyframe struct {
	Index     int
	Timestamp time.Duration
	Path      string
}

// Segment represents a transcript segment for output
type Segment struct {
	Start time.Duration
	End   time.Duration
	Text  string
}

// Result contains all data for markdown generation
type Result struct {
	InputPath   string
	Duration    time.Duration
	TotalFrames int
	Keyframes   []Keyframe
	Segments    []Segment
}

const markdownTemplate = `# Video Analysis: {{.Filename}}

## Metadata
- Duration: {{.DurationStr}}
- Original frames: {{.TotalFrames}}
- Keyframes extracted: {{.KeyframeCount}}
- Token estimate: ~{{.TokenEstimate}}

{{if .Segments}}
## Transcript

{{range .Segments}}[{{.StartStr}}] {{.Text}}
{{end}}
{{end}}
{{if .Keyframes}}
## Keyframes

{{range .Keyframes}}### Frame {{.Index}} ({{.TimestampStr}})
![Frame at {{.TimestampStr}}]({{.RelPath}})

{{end}}
{{end}}`

// templateData holds processed data for the template
type templateData struct {
	Filename      string
	DurationStr   string
	TotalFrames   int
	KeyframeCount int
	TokenEstimate int
	Segments      []segmentData
	Keyframes     []keyframeData
}

type segmentData struct {
	StartStr string
	Text     string
}

type keyframeData struct {
	Index        int
	TimestampStr string
	RelPath      string
}

// WriteMarkdown generates and writes the markdown output file
func WriteMarkdown(outputPath string, result Result) error {
	// Prepare template data
	data := templateData{
		Filename:      filepath.Base(result.InputPath),
		DurationStr:   formatDuration(result.Duration),
		TotalFrames:   result.TotalFrames,
		KeyframeCount: len(result.Keyframes),
		TokenEstimate: EstimateTokens(result),
	}

	// Process segments
	for _, seg := range result.Segments {
		data.Segments = append(data.Segments, segmentData{
			StartStr: formatDuration(seg.Start),
			Text:     strings.TrimSpace(seg.Text),
		})
	}

	// Process keyframes with relative paths
	outputDir := filepath.Dir(outputPath)
	for _, kf := range result.Keyframes {
		relPath, err := filepath.Rel(outputDir, kf.Path)
		if err != nil {
			relPath = kf.Path // Fall back to absolute path
		}
		data.Keyframes = append(data.Keyframes, keyframeData{
			Index:        kf.Index,
			TimestampStr: formatDuration(kf.Timestamp),
			RelPath:      relPath,
		})
	}

	// Parse and execute template
	tmpl, err := template.New("markdown").Parse(markdownTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return file.Close()
}

// formatDuration formats a duration as M:SS or H:MM:SS
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int64(d / time.Hour)
	d -= time.Duration(h) * time.Hour
	m := int64(d / time.Minute)
	d -= time.Duration(m) * time.Minute
	s := int64(d / time.Second)

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// EstimateTokens provides a rough estimate of tokens for the result
func EstimateTokens(result Result) int {
	// Rough estimates:
	// - ~1.3 tokens per word in transcript
	// - ~1000 tokens per image (varies by size/complexity, using conservative estimate)
	// - ~100 tokens for metadata/formatting

	var tokens int

	// Metadata overhead
	tokens += 100

	// Transcript tokens
	for _, seg := range result.Segments {
		words := len(strings.Fields(seg.Text))
		tokens += int(float64(words) * 1.3)
	}

	// Image tokens (conservative estimate for JPEG at quality 30, scaled 50%)
	tokens += len(result.Keyframes) * 1000

	return tokens
}
