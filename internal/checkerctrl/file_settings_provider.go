package checkerctrl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/proto"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

type FileSettingsProviderIn struct {
	fx.In

	Checkers []hazycheck.Checker `group:"checkers"`
	Config   *FileSettingsProviderConfig
}

type FileSettingsProviderConfig struct {
	Path string
}

var _ SettingsProvider = &FileSettingsProvider{}

type FileSettingsProvider struct {
	c        *FileSettingsProviderConfig
	checkers map[hazycheck.CheckerID]hazycheck.Checker
}

func NewFileSettingsProvider(in FileSettingsProviderIn) *FileSettingsProvider {
	checkers := make(map[hazycheck.CheckerID]hazycheck.Checker)
	for _, c := range in.Checkers {
		checkers[c.CheckerID()] = c
	}

	return &FileSettingsProvider{
		checkers: checkers,
		c:        in.Config,
	}
}

func (p *FileSettingsProvider) GetSettings(ctx context.Context) (*proto.Settings, error) {
	data, err := os.ReadFile(p.c.Path)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read data from file")
	}

	// First convert YAML to JSON to ensure proper format for protojson
	// This handles Duration fields and other special types correctly
	var yamlData any
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return nil, errors.Wrap(err, "cannot parse the config as yaml")
	}

	// Normalize duration strings to protojson-compatible format
	normalizeDurations(yamlData)

	// Convert to JSON bytes (preserves structure and handles duration strings)
	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		return nil, errors.Wrap(err, "cannot convert yaml to json")
	}

	// Unmarshal JSON into proto struct using protojson
	// protojson handles Duration fields in JSON format (e.g., "10s")
	protoS := &proto.Settings{}
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(jsonData, protoS); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal json into proto settings")
	}

	// Validate the settings
	if err := validateSettings(protoS); err != nil {
		return nil, errors.Wrap(err, "read the settings from file, but they were not valid")
	}

	return protoS, nil
}

// normalizeDurations recursively walks through the YAML data structure
// and converts duration strings to protojson-compatible format (seconds-based)
func normalizeDurations(data any) {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			// Check if this is a duration field (field name is "per" in Rate struct)
			if key == "per" {
				if strVal, ok := val.(string); ok {
					// Convert duration string to protojson format
					if converted := convertDurationString(strVal); converted != "" {
						v[key] = converted
					}
				}
			} else {
				// Recursively process nested structures
				normalizeDurations(val)
			}
		}
	case []any:
		for _, item := range v {
			normalizeDurations(item)
		}
	}
}

// convertDurationString converts Go duration string (e.g., "1m", "10s") to protojson format
// Protojson expects durations in seconds format like "60s", "10s", "60.5s", etc.
func convertDurationString(durStr string) string {
	dur, err := time.ParseDuration(durStr)
	if err != nil {
		// If it's not a valid duration, return empty string to leave it unchanged
		return ""
	}

	// Convert to total seconds (preserves fractional seconds)
	totalSeconds := dur.Seconds()
	if totalSeconds == 0 {
		return "0s"
	}

	// Format as "Xs" or "X.Ys" for protojson (preserves precision)
	if totalSeconds == float64(int64(totalSeconds)) {
		// Integer seconds
		return fmt.Sprintf("%ds", int64(totalSeconds))
	}
	// Fractional seconds
	return fmt.Sprintf("%gs", totalSeconds)
}
