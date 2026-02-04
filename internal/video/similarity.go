package video

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

const (
	// Comparison dimensions for faster processing
	compWidth  = 200
	compHeight = 400
)

// DetectKeyframes analyzes frames and returns those that differ significantly
// from their predecessors based on normalized cross-correlation
func DetectKeyframes(frames []Frame, threshold float64, onProgress ProgressFunc) ([]Keyframe, error) {
	if len(frames) == 0 {
		return nil, nil
	}

	var keyframes []Keyframe

	// Always include first frame
	keyframes = append(keyframes, Keyframe{
		Path:      frames[0].Path,
		Index:     frames[0].Index,
		Timestamp: frames[0].Timestamp,
	})

	if len(frames) == 1 {
		if onProgress != nil {
			onProgress(1.0)
		}
		return keyframes, nil
	}

	// Load and process first frame
	prevGray, err := loadAndProcessFrame(frames[0].Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load first frame: %w", err)
	}

	total := len(frames) - 1

	// Compare consecutive frames
	for i := 1; i < len(frames); i++ {
		currGray, err := loadAndProcessFrame(frames[i].Path)
		if err != nil {
			return nil, fmt.Errorf("failed to load frame %d: %w", i, err)
		}

		// Compute normalized cross-correlation
		correlation := normalizedCrossCorrelation(prevGray, currGray)

		// If correlation is below threshold, this is a keyframe (significant change)
		if correlation < threshold {
			keyframes = append(keyframes, Keyframe{
				Path:      frames[i].Path,
				Index:     frames[i].Index,
				Timestamp: frames[i].Timestamp,
			})
		}

		prevGray = currGray

		if onProgress != nil {
			onProgress(float64(i) / float64(total))
		}
	}

	// Always include last frame if not already included
	lastFrame := frames[len(frames)-1]
	if len(keyframes) == 0 || keyframes[len(keyframes)-1].Index != lastFrame.Index {
		keyframes = append(keyframes, Keyframe(lastFrame))
	}

	return keyframes, nil
}

// loadAndProcessFrame loads an image, resizes it, and converts to grayscale
func loadAndProcessFrame(path string) ([]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var img image.Image

	ext := filepath.Ext(path)
	switch ext {
	case ".png":
		img, err = png.Decode(file)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}

	if err != nil {
		return nil, err
	}

	// Resize for faster comparison
	resized := resize.Resize(compWidth, compHeight, img, resize.Bilinear)

	// Convert to grayscale values
	bounds := resized.Bounds()
	gray := make([]float64, bounds.Dx()*bounds.Dy())

	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := resized.At(x, y).RGBA()
			// Standard grayscale conversion
			luminance := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
			gray[idx] = luminance / 65535.0 // Normalize to 0-1
			idx++
		}
	}

	return gray, nil
}

// normalizedCrossCorrelation computes NCC between two grayscale images
// Returns a value between -1 and 1, where 1 means identical
func normalizedCrossCorrelation(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	n := float64(len(a))

	// Compute means
	var sumA, sumB float64
	for i := range a {
		sumA += a[i]
		sumB += b[i]
	}
	meanA := sumA / n
	meanB := sumB / n

	// Compute standard deviations and cross-correlation
	var sumProduct, sumSqA, sumSqB float64
	for i := range a {
		diffA := a[i] - meanA
		diffB := b[i] - meanB
		sumProduct += diffA * diffB
		sumSqA += diffA * diffA
		sumSqB += diffB * diffB
	}

	stdA := math.Sqrt(sumSqA / n)
	stdB := math.Sqrt(sumSqB / n)

	// Avoid division by zero
	if stdA < 1e-10 || stdB < 1e-10 {
		return 1.0 // Both images are essentially constant
	}

	// Normalized cross-correlation
	ncc := sumProduct / (n * stdA * stdB)

	return ncc
}

// SaveKeyframes saves keyframes as JPEGs with optional scaling and quality settings
func SaveKeyframes(keyframes []Keyframe, outputDir string, quality int, scale float64, onProgress ProgressFunc) error {
	total := len(keyframes)
	for i, kf := range keyframes {
		if err := saveKeyframe(kf, outputDir, quality, scale); err != nil {
			return err
		}
		if onProgress != nil {
			onProgress(float64(i+1) / float64(total))
		}
	}
	return nil
}

func saveKeyframe(kf Keyframe, outputDir string, quality int, scale float64) error {
	// Load original frame
	file, err := os.Open(kf.Path)
	if err != nil {
		return fmt.Errorf("failed to open frame %d: %w", kf.Index, err)
	}
	defer func() { _ = file.Close() }()

	var img image.Image
	ext := filepath.Ext(kf.Path)
	switch ext {
	case ".png":
		img, err = png.Decode(file)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	default:
		return fmt.Errorf("unsupported image format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to decode frame %d: %w", kf.Index, err)
	}

	// Scale if needed
	if scale != 1.0 {
		bounds := img.Bounds()
		newWidth := uint(float64(bounds.Dx()) * scale)
		newHeight := uint(float64(bounds.Dy()) * scale)
		img = resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
	}

	// Save as JPEG
	outputPath := filepath.Join(outputDir, fmt.Sprintf("frame_%04d.jpg", kf.Index))
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	if err := jpeg.Encode(outFile, img, &jpeg.Options{Quality: quality}); err != nil {
		return fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return outFile.Close()
}
