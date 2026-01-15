package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ExportFormat represents the format for export
type ExportFormat string

const (
	ExportJSON ExportFormat = "json"
	ExportYAML ExportFormat = "yaml"
)

// Exporter handles exporting data to files
type Exporter struct {
	outputDir string
}

// NewExporter creates a new exporter
func NewExporter(outputDir string) *Exporter {
	if outputDir == "" {
		outputDir = "."
	}
	return &Exporter{outputDir: outputDir}
}

// Export exports data to a file
func (e *Exporter) Export(data interface{}, resourceType, resourceID string, format ExportFormat) (string, error) {
	// Create filename
	timestamp := time.Now().Format("20060102-150405")
	ext := string(format)
	if format == ExportYAML {
		ext = "yaml"
	}

	// Sanitize resource ID for filename
	safeID := sanitizeFilename(resourceID)
	filename := fmt.Sprintf("%s-%s-%s.%s", resourceType, safeID, timestamp, ext)
	filepath := filepath.Join(e.outputDir, filename)

	// Marshal data
	var content []byte
	var err error

	switch format {
	case ExportJSON:
		content, err = json.MarshalIndent(data, "", "  ")
	case ExportYAML:
		content, err = yaml.Marshal(data)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filepath, nil
}

// ExportList exports multiple resources to a file
func (e *Exporter) ExportList(data interface{}, resourceType string, count int, format ExportFormat) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	ext := string(format)
	if format == ExportYAML {
		ext = "yaml"
	}

	filename := fmt.Sprintf("%s-list-%d-%s.%s", resourceType, count, timestamp, ext)
	filepath := filepath.Join(e.outputDir, filename)

	var content []byte
	var err error

	switch format {
	case ExportJSON:
		content, err = json.MarshalIndent(data, "", "  ")
	case ExportYAML:
		content, err = yaml.Marshal(data)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(filepath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filepath, nil
}

// ToJSON converts data to JSON string
func ToJSON(data interface{}) (string, error) {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ToYAML converts data to YAML string
func ToYAML(data interface{}) (string, error) {
	content, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// sanitizeFilename removes or replaces characters that are not safe for filenames
func sanitizeFilename(s string) string {
	// Replace common problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
		" ", "_",
	)
	result := replacer.Replace(s)

	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}

	return result
}
