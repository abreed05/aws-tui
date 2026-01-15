package components

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// ClipboardCopiedMsg is sent when content is copied to clipboard
type ClipboardCopiedMsg struct {
	Content string
	Label   string
	Success bool
	Error   error
}

// CopyToClipboard copies text to the system clipboard
func CopyToClipboard(content, label string) tea.Cmd {
	return func() tea.Msg {
		err := writeToClipboard(content)
		if err != nil {
			return ClipboardCopiedMsg{
				Content: content,
				Label:   label,
				Success: false,
				Error:   err,
			}
		}
		return ClipboardCopiedMsg{
			Content: content,
			Label:   label,
			Success: true,
		}
	}
}

// CopyJSONToClipboard copies a map as formatted JSON to clipboard
func CopyJSONToClipboard(data map[string]interface{}, label string) tea.Cmd {
	return func() tea.Msg {
		jsonBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return ClipboardCopiedMsg{
				Label:   label,
				Success: false,
				Error:   fmt.Errorf("failed to marshal JSON: %w", err),
			}
		}

		content := string(jsonBytes)
		err = writeToClipboard(content)
		if err != nil {
			return ClipboardCopiedMsg{
				Content: content,
				Label:   label,
				Success: false,
				Error:   err,
			}
		}
		return ClipboardCopiedMsg{
			Content: content,
			Label:   label,
			Success: true,
		}
	}
}

// writeToClipboard writes content to the system clipboard using OS-specific tools
func writeToClipboard(content string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel, then wl-copy (Wayland)
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start clipboard command: %w", err)
	}

	if _, err := stdin.Write([]byte(content)); err != nil {
		return fmt.Errorf("failed to write to clipboard: %w", err)
	}

	if err := stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("clipboard command failed: %w", err)
	}

	return nil
}
