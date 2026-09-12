// Package config handles iradio's on-disk configuration and persistence.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Dir returns the iradio config directory, creating it on first use.
func Dir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	dir := filepath.Join(configDir, "iradio")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func path(name string) string { return filepath.Join(Dir(), name) }

// LoadBoolMap reads a JSON map[string]bool, returning an empty map on error.
func LoadBoolMap(name string) map[string]bool {
	m := make(map[string]bool)
	if data, err := os.ReadFile(path(name)); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	return m
}

// SaveBoolMap writes a map[string]bool as indented JSON.
func SaveBoolMap(name string, m map[string]bool) {
	if data, err := json.MarshalIndent(m, "", "  "); err == nil {
		_ = os.WriteFile(path(name), data, 0644)
	}
}

// LoadIntMap reads a JSON map[string]int, returning an empty map on error.
func LoadIntMap(name string) map[string]int {
	m := make(map[string]int)
	if data, err := os.ReadFile(path(name)); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	return m
}

// SaveIntMap writes a map[string]int as indented JSON.
func SaveIntMap(name string, m map[string]int) {
	if data, err := json.MarshalIndent(m, "", "  "); err == nil {
		_ = os.WriteFile(path(name), data, 0644)
	}
}

// LoadString reads a plain string value, returning "" on error.
func LoadString(name string) string {
	data, err := os.ReadFile(path(name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SaveString writes a plain string value to the config directory.
func SaveString(name, val string) {
	_ = os.WriteFile(path(name), []byte(val), 0644)
}

// LoadInt reads an integer value from the config directory, returning fallback on error.
func LoadInt(name string, fallback int) int {
	data, err := os.ReadFile(path(name))
	if err != nil {
		return fallback
	}
	var val int
	if err := json.Unmarshal(data, &val); err == nil {
		return val
	}
	if n, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
		return n
	}
	return fallback
}

// SaveInt writes an integer value as JSON to the config directory.
func SaveInt(name string, val int) {
	if data, err := json.Marshal(val); err == nil {
		_ = os.WriteFile(path(name), data, 0644)
	}
}

// WriteExampleStations writes a starter stations.example.json if none exists.
func WriteExampleStations() {
	exampleFile := path("stations.example.json")
	if _, err := os.Stat(exampleFile); os.IsNotExist(err) {
		exampleContent := `[
  {
    "id": "MYSTATION",
    "region": "TW",
    "name_zh": "自訂電台範例",
    "name_en": "My Custom Station",
    "dial": "Online",
    "desc": "Custom stream description",
    "stream_url": "https://example.com/stream.m3u8"
  }
]
`
		_ = os.WriteFile(exampleFile, []byte(exampleContent), 0644)
	}
}

// SystemVolume returns the current system volume as a percentage string.
func SystemVolume() string {
	if out, err := exec.Command("amixer", "sget", "Master").Output(); err == nil {
		s := string(out)
		if idx := strings.Index(s, "["); idx != -1 {
			if end := strings.Index(s[idx:], "%]"); end != -1 {
				return s[idx+1 : idx+end+1]
			}
		}
	}
	if out, err := exec.Command("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@").Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) >= 2 {
			if v, err := strconv.ParseFloat(fields[1], 64); err == nil {
				return fmt.Sprintf("%d%%", int(v*100))
			}
		}
	}
	return "63%"
}
