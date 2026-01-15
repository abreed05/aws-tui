package styles

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// ThemeConfig represents a theme configuration loaded from YAML
type ThemeConfig struct {
	Name   string       `yaml:"name"`
	Colors ColorsConfig `yaml:"colors"`
}

// ColorsConfig represents color configuration in YAML
type ColorsConfig struct {
	Primary     string `yaml:"primary"`
	Secondary   string `yaml:"secondary"`
	Accent      string `yaml:"accent"`
	Background  string `yaml:"background"`
	Foreground  string `yaml:"foreground"`
	Muted       string `yaml:"muted"`
	Success     string `yaml:"success"`
	Warning     string `yaml:"warning"`
	Error       string `yaml:"error"`
	Info        string `yaml:"info"`
	Border      string `yaml:"border"`
	Selection   string `yaml:"selection"`
	SelectionFg string `yaml:"selection_fg"`
}

// Built-in theme color palettes
var builtinThemes = map[string]ColorsConfig{
	"default": {
		Primary:     "39",
		Secondary:   "62",
		Accent:      "212",
		Background:  "235",
		Foreground:  "252",
		Muted:       "245",
		Success:     "78",
		Warning:     "214",
		Error:       "196",
		Info:        "81",
		Border:      "240",
		Selection:   "57",
		SelectionFg: "229",
	},
	"dark": {
		Primary:     "75",
		Secondary:   "141",
		Accent:      "219",
		Background:  "232",
		Foreground:  "255",
		Muted:       "244",
		Success:     "114",
		Warning:     "220",
		Error:       "203",
		Info:        "117",
		Border:      "238",
		Selection:   "24",
		SelectionFg: "255",
	},
	"light": {
		Primary:     "27",
		Secondary:   "91",
		Accent:      "162",
		Background:  "255",
		Foreground:  "235",
		Muted:       "245",
		Success:     "28",
		Warning:     "172",
		Error:       "160",
		Info:        "33",
		Border:      "250",
		Selection:   "153",
		SelectionFg: "235",
	},
	"nord": {
		Primary:     "110",
		Secondary:   "139",
		Accent:      "180",
		Background:  "236",
		Foreground:  "253",
		Muted:       "245",
		Success:     "108",
		Warning:     "222",
		Error:       "174",
		Info:        "110",
		Border:      "239",
		Selection:   "60",
		SelectionFg: "253",
	},
	"dracula": {
		Primary:     "141",
		Secondary:   "212",
		Accent:      "219",
		Background:  "235",
		Foreground:  "253",
		Muted:       "245",
		Success:     "84",
		Warning:     "228",
		Error:       "210",
		Info:        "117",
		Border:      "239",
		Selection:   "61",
		SelectionFg: "253",
	},
}

// LoadTheme loads a theme by name, checking built-in themes first, then custom files
func LoadTheme(name string, configDir string) (Theme, error) {
	// Check built-in themes first
	if colors, ok := builtinThemes[name]; ok {
		return NewThemeFromColors(colors), nil
	}

	// Try to load from custom theme file
	themesDir := filepath.Join(configDir, "themes")
	themePath := filepath.Join(themesDir, name+".yaml")

	theme, err := LoadThemeFromFile(themePath)
	if err != nil {
		// Fallback to default theme
		return DefaultTheme(), err
	}

	return theme, nil
}

// LoadThemeFromFile loads a theme from a YAML file
func LoadThemeFromFile(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, err
	}

	var config ThemeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Theme{}, err
	}

	return NewThemeFromColors(config.Colors), nil
}

// NewThemeFromColors creates a Theme from a ColorsConfig
func NewThemeFromColors(cfg ColorsConfig) Theme {
	c := Colors{
		Primary:     lipgloss.Color(cfg.Primary),
		Secondary:   lipgloss.Color(cfg.Secondary),
		Accent:      lipgloss.Color(cfg.Accent),
		Background:  lipgloss.Color(cfg.Background),
		Foreground:  lipgloss.Color(cfg.Foreground),
		Muted:       lipgloss.Color(cfg.Muted),
		Success:     lipgloss.Color(cfg.Success),
		Warning:     lipgloss.Color(cfg.Warning),
		Error:       lipgloss.Color(cfg.Error),
		Info:        lipgloss.Color(cfg.Info),
		Border:      lipgloss.Color(cfg.Border),
		Selection:   lipgloss.Color(cfg.Selection),
		SelectionFg: lipgloss.Color(cfg.SelectionFg),
	}

	return Theme{
		Colors: c,

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.Foreground).
			Background(c.Primary).
			Padding(0, 1),

		Footer: lipgloss.NewStyle().
			Foreground(c.Muted).
			Background(lipgloss.Color("236")).
			Padding(0, 1),

		Breadcrumb: lipgloss.NewStyle().
			Foreground(c.Muted).
			Padding(0, 1),

		Table: TableStyles{
			Header: lipgloss.NewStyle().
				Bold(true).
				Foreground(c.Primary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(c.Border).
				BorderBottom(true),
			Row: lipgloss.NewStyle().
				Foreground(c.Foreground),
			Selected: lipgloss.NewStyle().
				Bold(true).
				Foreground(c.SelectionFg).
				Background(c.Selection),
			Cell: lipgloss.NewStyle().
				Padding(0, 1),
		},

		Detail: DetailStyles{
			Section: lipgloss.NewStyle().
				Bold(true).
				Foreground(c.Accent).
				MarginTop(1),
			Key: lipgloss.NewStyle().
				Foreground(c.Info),
			Value: lipgloss.NewStyle().
				Foreground(c.Foreground),
			Border: lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(c.Border).
				Padding(1, 2),
		},

		Search: lipgloss.NewStyle().
			Foreground(c.Foreground).
			Background(lipgloss.Color("237")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.Primary),

		Command: lipgloss.NewStyle().
			Foreground(c.Foreground).
			Background(lipgloss.Color("237")).
			Padding(0, 1),

		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.Secondary).
			Padding(1, 2),

		Help: lipgloss.NewStyle().
			Foreground(c.Muted),

		StatusBar: lipgloss.NewStyle().
			Foreground(c.Foreground).
			Background(lipgloss.Color("236")),

		ErrorMessage: lipgloss.NewStyle().
			Foreground(c.Error).
			Bold(true),
	}
}

// AvailableThemes returns a list of built-in theme names
func AvailableThemes() []string {
	themes := make([]string, 0, len(builtinThemes))
	for name := range builtinThemes {
		themes = append(themes, name)
	}
	return themes
}
