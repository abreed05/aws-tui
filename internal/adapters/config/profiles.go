package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

// Profile represents an AWS profile
type Profile struct {
	Name       string
	Region     string
	RoleARN    string
	SourceProf string
	SSOStartURL string
	SSORegion  string
	SSOAccount string
	SSORoleName string
	IsSSO      bool
}

// Region represents an AWS region
type Region struct {
	Name        string
	Description string
}

// ProfileLoader discovers AWS profiles from config files
type ProfileLoader struct {
	configPath      string
	credentialsPath string
}

// NewProfileLoader creates a profile loader with default paths
func NewProfileLoader() *ProfileLoader {
	home, _ := os.UserHomeDir()
	return &ProfileLoader{
		configPath:      filepath.Join(home, ".aws", "config"),
		credentialsPath: filepath.Join(home, ".aws", "credentials"),
	}
}

// ListProfiles returns all available AWS profiles
func (pl *ProfileLoader) ListProfiles() ([]Profile, error) {
	profiles := make(map[string]*Profile)

	// Parse config file (has profile settings like region, role_arn, sso)
	if cfg, err := ini.Load(pl.configPath); err == nil {
		for _, section := range cfg.Sections() {
			name := section.Name()
			if name == "DEFAULT" {
				continue
			}

			// Config file uses "profile <name>" prefix, except for default
			profileName := strings.TrimPrefix(name, "profile ")

			p := &Profile{
				Name:        profileName,
				Region:      section.Key("region").String(),
				RoleARN:     section.Key("role_arn").String(),
				SourceProf:  section.Key("source_profile").String(),
				SSOStartURL: section.Key("sso_start_url").String(),
				SSORegion:   section.Key("sso_region").String(),
				SSOAccount:  section.Key("sso_account_id").String(),
				SSORoleName: section.Key("sso_role_name").String(),
			}
			p.IsSSO = p.SSOStartURL != ""

			profiles[profileName] = p
		}
	}

	// Parse credentials file (adds credential-only profiles)
	if creds, err := ini.Load(pl.credentialsPath); err == nil {
		for _, section := range creds.Sections() {
			name := section.Name()
			if name == "DEFAULT" {
				continue
			}

			if _, exists := profiles[name]; !exists {
				profiles[name] = &Profile{Name: name}
			}
		}
	}

	// Check for AWS_PROFILE environment variable and ensure that profile exists in list
	if envProfile := os.Getenv("AWS_PROFILE"); envProfile != "" {
		if _, exists := profiles[envProfile]; !exists {
			profiles[envProfile] = &Profile{
				Name:   envProfile,
				Region: os.Getenv("AWS_REGION"),
			}
		}
	}

	// Always ensure "default" profile exists if we have any credentials
	// This catches cases where credentials are provided via env vars
	if _, exists := profiles["default"]; !exists {
		// Check if we have env var credentials
		if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_SESSION_TOKEN") != "" {
			profiles["default"] = &Profile{
				Name:   "default",
				Region: os.Getenv("AWS_REGION"),
			}
		}
	}

	// Convert to slice and sort
	result := make([]Profile, 0, len(profiles))
	for _, p := range profiles {
		result = append(result, *p)
	}

	sort.Slice(result, func(i, j int) bool {
		// Put "default" first
		if result[i].Name == "default" {
			return true
		}
		if result[j].Name == "default" {
			return false
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// GetProfile returns a specific profile by name
func (pl *ProfileLoader) GetProfile(name string) (*Profile, error) {
	profiles, err := pl.ListProfiles()
	if err != nil {
		return nil, err
	}

	for _, p := range profiles {
		if p.Name == name {
			return &p, nil
		}
	}

	return nil, nil
}

// ListRegions returns all AWS regions
func (pl *ProfileLoader) ListRegions() []Region {
	return []Region{
		{Name: "us-east-1", Description: "US East (N. Virginia)"},
		{Name: "us-east-2", Description: "US East (Ohio)"},
		{Name: "us-west-1", Description: "US West (N. California)"},
		{Name: "us-west-2", Description: "US West (Oregon)"},
		{Name: "af-south-1", Description: "Africa (Cape Town)"},
		{Name: "ap-east-1", Description: "Asia Pacific (Hong Kong)"},
		{Name: "ap-south-1", Description: "Asia Pacific (Mumbai)"},
		{Name: "ap-south-2", Description: "Asia Pacific (Hyderabad)"},
		{Name: "ap-southeast-1", Description: "Asia Pacific (Singapore)"},
		{Name: "ap-southeast-2", Description: "Asia Pacific (Sydney)"},
		{Name: "ap-southeast-3", Description: "Asia Pacific (Jakarta)"},
		{Name: "ap-southeast-4", Description: "Asia Pacific (Melbourne)"},
		{Name: "ap-northeast-1", Description: "Asia Pacific (Tokyo)"},
		{Name: "ap-northeast-2", Description: "Asia Pacific (Seoul)"},
		{Name: "ap-northeast-3", Description: "Asia Pacific (Osaka)"},
		{Name: "ca-central-1", Description: "Canada (Central)"},
		{Name: "eu-central-1", Description: "Europe (Frankfurt)"},
		{Name: "eu-central-2", Description: "Europe (Zurich)"},
		{Name: "eu-west-1", Description: "Europe (Ireland)"},
		{Name: "eu-west-2", Description: "Europe (London)"},
		{Name: "eu-west-3", Description: "Europe (Paris)"},
		{Name: "eu-south-1", Description: "Europe (Milan)"},
		{Name: "eu-south-2", Description: "Europe (Spain)"},
		{Name: "eu-north-1", Description: "Europe (Stockholm)"},
		{Name: "il-central-1", Description: "Israel (Tel Aviv)"},
		{Name: "me-south-1", Description: "Middle East (Bahrain)"},
		{Name: "me-central-1", Description: "Middle East (UAE)"},
		{Name: "sa-east-1", Description: "South America (São Paulo)"},
	}
}
