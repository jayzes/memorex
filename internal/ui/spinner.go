// Package ui provides terminal UI components for progress indication.
package ui

import (
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Styles
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Progress bar characters
	progressFull  = successStyle.Render("█")
	progressEmpty = dimStyle.Render("░")
)

// Step represents a single step in a multi-step process.
type Step struct {
	Name     string
	mu       sync.Mutex
	percent  float64
	complete bool
	failed   bool
}

// NewStep creates a new step with the given name and starts displaying it.
func NewStep(name string) *Step {
	s := &Step{Name: name}
	s.render()
	return s
}

// Update updates the step's progress (0.0 to 1.0).
func (s *Step) Update(percent float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.complete || s.failed {
		return
	}
	s.percent = percent
	s.render()
}

// Complete marks the step as successfully completed.
func (s *Step) Complete(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.complete = true
	s.percent = 1.0
	fmt.Fprintf(os.Stderr, "\r\033[K%s %s\n",
		successStyle.Render("✓"),
		textStyle.Render(message))
}

// Error marks the step as failed.
func (s *Step) Error(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed = true
	fmt.Fprintf(os.Stderr, "\r\033[K%s %s\n",
		errorStyle.Render("✗"),
		textStyle.Render(message))
}

func (s *Step) render() {
	pct := s.percent * 100
	if pct > 100 {
		pct = 100
	}
	bar := renderProgressBar(pct, 20)
	fmt.Fprintf(os.Stderr, "\r\033[K%s %s %s",
		spinnerStyle.Render("→"),
		textStyle.Render(s.Name),
		dimStyle.Render(fmt.Sprintf("%s %3.0f%%", bar, pct)))
}

func renderProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += progressFull
		} else {
			bar += progressEmpty
		}
	}
	return bar
}

// PrintHeader prints a styled header.
func PrintHeader(title string) {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	fmt.Fprintln(os.Stderr, headerStyle.Render(title))
}

// PrintInfo prints an info message.
func PrintInfo(message string) {
	fmt.Fprintln(os.Stderr, dimStyle.Render("  ")+textStyle.Render(message))
}

// PrintSuccess prints a success message.
func PrintSuccess(message string) {
	fmt.Fprintln(os.Stderr, successStyle.Render("✓ ")+textStyle.Render(message))
}

// PrintWarning prints a warning message.
func PrintWarning(message string) {
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	fmt.Fprintln(os.Stderr, warnStyle.Render("⚠ ")+textStyle.Render(message))
}

// PrintError prints an error message.
func PrintError(message string) {
	fmt.Fprintln(os.Stderr, errorStyle.Render("✗ ")+textStyle.Render(message))
}
