package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Bookmark represents a saved resource reference
type Bookmark struct {
	Name         string    `yaml:"name"`
	ResourceType string    `yaml:"resource_type"`
	ResourceID   string    `yaml:"resource_id"`
	ARN          string    `yaml:"arn"`
	Region       string    `yaml:"region"`
	Profile      string    `yaml:"profile"`
	CreatedAt    time.Time `yaml:"created_at"`
}

// BookmarkStore manages bookmark persistence
type BookmarkStore struct {
	filepath  string
	bookmarks []Bookmark
}

// NewBookmarkStore creates a new bookmark store
func NewBookmarkStore() *BookmarkStore {
	configDir := getConfigDir()
	return &BookmarkStore{
		filepath:  filepath.Join(configDir, "bookmarks.yaml"),
		bookmarks: []Bookmark{},
	}
}

// getConfigDir returns the config directory path
func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".aws-tui"
	}
	return filepath.Join(home, ".config", "aws-tui")
}

// Load loads bookmarks from disk
func (s *BookmarkStore) Load() error {
	// Ensure config directory exists
	dir := filepath.Dir(s.filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(s.filepath); os.IsNotExist(err) {
		s.bookmarks = []Bookmark{}
		return nil
	}

	data, err := os.ReadFile(s.filepath)
	if err != nil {
		return fmt.Errorf("failed to read bookmarks file: %w", err)
	}

	var bookmarks []Bookmark
	if err := yaml.Unmarshal(data, &bookmarks); err != nil {
		return fmt.Errorf("failed to parse bookmarks file: %w", err)
	}

	s.bookmarks = bookmarks
	return nil
}

// Save saves bookmarks to disk
func (s *BookmarkStore) Save() error {
	// Ensure config directory exists
	dir := filepath.Dir(s.filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(s.bookmarks)
	if err != nil {
		return fmt.Errorf("failed to marshal bookmarks: %w", err)
	}

	if err := os.WriteFile(s.filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write bookmarks file: %w", err)
	}

	return nil
}

// Add adds a new bookmark
func (s *BookmarkStore) Add(bookmark Bookmark) error {
	// Check for duplicates
	for i, b := range s.bookmarks {
		if b.ResourceType == bookmark.ResourceType && b.ResourceID == bookmark.ResourceID {
			// Update existing bookmark
			s.bookmarks[i] = bookmark
			return s.Save()
		}
	}

	bookmark.CreatedAt = time.Now()
	s.bookmarks = append(s.bookmarks, bookmark)
	return s.Save()
}

// Remove removes a bookmark by index
func (s *BookmarkStore) Remove(index int) error {
	if index < 0 || index >= len(s.bookmarks) {
		return fmt.Errorf("bookmark index out of range")
	}

	s.bookmarks = append(s.bookmarks[:index], s.bookmarks[index+1:]...)
	return s.Save()
}

// RemoveByID removes a bookmark by resource type and ID
func (s *BookmarkStore) RemoveByID(resourceType, resourceID string) error {
	for i, b := range s.bookmarks {
		if b.ResourceType == resourceType && b.ResourceID == resourceID {
			return s.Remove(i)
		}
	}
	return nil
}

// List returns all bookmarks
func (s *BookmarkStore) List() []Bookmark {
	return s.bookmarks
}

// Get returns a bookmark by index
func (s *BookmarkStore) Get(index int) (Bookmark, bool) {
	if index < 0 || index >= len(s.bookmarks) {
		return Bookmark{}, false
	}
	return s.bookmarks[index], true
}

// IsBookmarked checks if a resource is bookmarked
func (s *BookmarkStore) IsBookmarked(resourceType, resourceID string) bool {
	for _, b := range s.bookmarks {
		if b.ResourceType == resourceType && b.ResourceID == resourceID {
			return true
		}
	}
	return false
}

// Count returns the number of bookmarks
func (s *BookmarkStore) Count() int {
	return len(s.bookmarks)
}
