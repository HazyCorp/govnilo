package settingsprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/stretchr/testify/require"
)

func TestFileSettingsProvider_ValidYAML(t *testing.T) {
	yamlContent := `
services:
  - name: example
    target: http://example.com:8080
    checkers:
      - name: checker1
        run_options:
          rate:
            times: 10
            per: 10s
          max_goroutines: 100
        success_points: 1.0
        fail_penalty: 5.0
      - name: checker2
        run_options:
          rate:
            times: 5
            per: 1m
          max_goroutines: 50
        success_points: 2.0
        fail_penalty: 10.0
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "settings.yaml")
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	provider := NewFile(FileIn{
		Config: &FileConfig{
			Path: filePath,
		},
		Checkers: []hazycheck.Checker{},
	})

	ctx := context.Background()
	settings, err := provider.GetSettings(ctx)
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.Len(t, settings.GetServices(), 1)

	service := settings.GetServices()[0]
	require.Equal(t, "example", service.GetName())
	require.Equal(t, "http://example.com:8080", service.GetTarget())
	require.Len(t, service.GetCheckers(), 2)

	checker1 := service.GetCheckers()[0]
	require.Equal(t, "checker1", checker1.GetName())
	require.Equal(t, 1.0, checker1.GetSuccessPoints())
	require.Equal(t, 5.0, checker1.GetFailPenalty())
	require.NotNil(t, checker1.GetRunOptions())
	require.NotNil(t, checker1.GetRunOptions().GetRate())
	require.Equal(t, uint64(10), checker1.GetRunOptions().GetRate().GetTimes())
	require.Equal(t, time.Second*10, checker1.GetRunOptions().GetRate().GetPer().AsDuration())
	require.Equal(t, int32(100), checker1.GetRunOptions().GetMaxGoroutines())

	checker2 := service.GetCheckers()[1]
	require.Equal(t, "checker2", checker2.GetName())
	require.Equal(t, 2.0, checker2.GetSuccessPoints())
	require.Equal(t, 10.0, checker2.GetFailPenalty())
	require.Equal(t, uint64(5), checker2.GetRunOptions().GetRate().GetTimes())
	require.Equal(t, time.Minute, checker2.GetRunOptions().GetRate().GetPer().AsDuration())
	require.Equal(t, int32(50), checker2.GetRunOptions().GetMaxGoroutines())
}

func TestFileSettingsProvider_MultipleServices(t *testing.T) {
	yamlContent := `
services:
  - name: service1
    target: http://service1.com:8080
    checkers:
      - name: checker1
        run_options:
          rate:
            times: 1
            per: 1s
          max_goroutines: 10
        success_points: 1.0
        fail_penalty: 1.0
  - name: service2
    target: http://service2.com:8080
    checkers:
      - name: checker2
        run_options:
          rate:
            times: 2
            per: 2s
          max_goroutines: 20
        success_points: 2.0
        fail_penalty: 2.0
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "settings.yaml")
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	provider := NewFile(FileIn{
		Config: &FileConfig{
			Path: filePath,
		},
		Checkers: []hazycheck.Checker{},
	})

	ctx := context.Background()
	settings, err := provider.GetSettings(ctx)
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.Len(t, settings.GetServices(), 2)

	service1 := settings.GetServices()[0]
	require.Equal(t, "service1", service1.GetName())
	require.Equal(t, "http://service1.com:8080", service1.GetTarget())
	require.Len(t, service1.GetCheckers(), 1)

	service2 := settings.GetServices()[1]
	require.Equal(t, "service2", service2.GetName())
	require.Equal(t, "http://service2.com:8080", service2.GetTarget())
	require.Len(t, service2.GetCheckers(), 1)
}

func TestFileSettingsProvider_InvalidYAML(t *testing.T) {
	yamlContent := `
services:
  - name: example
    target: http://example.com:8080
    checkers:
      - name: checker1
        invalid_field: value
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "settings.yaml")
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	provider := NewFile(FileIn{
		Config: &FileConfig{
			Path: filePath,
		},
		Checkers: []hazycheck.Checker{},
	})

	ctx := context.Background()
	settings, err := provider.GetSettings(ctx)
	// Invalid YAML structure should fail validation or parsing
	require.Error(t, err)
	require.Nil(t, settings)
}

func TestFileSettingsProvider_InvalidDuration(t *testing.T) {
	yamlContent := `
services:
  - name: example
    target: http://example.com:8080
    checkers:
      - name: checker1
        run_options:
          rate:
            times: 10
            per: 0s
          max_goroutines: 100
        success_points: 1.0
        fail_penalty: 5.0
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "settings.yaml")
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	provider := NewFile(FileIn{
		Config: &FileConfig{
			Path: filePath,
		},
		Checkers: []hazycheck.Checker{},
	})

	ctx := context.Background()
	settings, err := provider.GetSettings(ctx)
	require.Error(t, err)
	require.Nil(t, settings)
}

func TestFileSettingsProvider_MissingFile(t *testing.T) {
	provider := NewFile(FileIn{
		Config: &FileConfig{
			Path: "/nonexistent/path/settings.yaml",
		},
		Checkers: []hazycheck.Checker{},
	})

	ctx := context.Background()
	settings, err := provider.GetSettings(ctx)
	require.Error(t, err)
	require.Nil(t, settings)
}

func TestFileSettingsProvider_DuplicateServiceNames(t *testing.T) {
	yamlContent := `
services:
  - name: example
    target: http://example.com:8080
    checkers:
      - name: checker1
        run_options:
          rate:
            times: 10
            per: 10s
          max_goroutines: 100
        success_points: 1.0
        fail_penalty: 5.0
  - name: example
    target: http://example2.com:8080
    checkers:
      - name: checker2
        run_options:
          rate:
            times: 5
            per: 1m
          max_goroutines: 50
        success_points: 2.0
        fail_penalty: 10.0
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "settings.yaml")
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	provider := NewFile(FileIn{
		Config: &FileConfig{
			Path: filePath,
		},
		Checkers: []hazycheck.Checker{},
	})

	ctx := context.Background()
	settings, err := provider.GetSettings(ctx)
	require.Error(t, err)
	require.Nil(t, settings)
}

func TestFileSettingsProvider_DuplicateCheckerNames(t *testing.T) {
	yamlContent := `
services:
  - name: example
    target: http://example.com:8080
    checkers:
      - name: checker1
        run_options:
          rate:
            times: 10
            per: 10s
          max_goroutines: 100
        success_points: 1.0
        fail_penalty: 5.0
      - name: checker1
        run_options:
          rate:
            times: 5
            per: 1m
          max_goroutines: 50
        success_points: 2.0
        fail_penalty: 10.0
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "settings.yaml")
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	provider := NewFile(FileIn{
		Config: &FileConfig{
			Path: filePath,
		},
		Checkers: []hazycheck.Checker{},
	})

	ctx := context.Background()
	settings, err := provider.GetSettings(ctx)
	require.Error(t, err)
	require.Nil(t, settings)
}

func TestFileSettingsProvider_EmptyServiceName(t *testing.T) {
	yamlContent := `
services:
  - name: ""
    target: http://example.com:8080
    checkers:
      - name: checker1
        run_options:
          rate:
            times: 10
            per: 10s
          max_goroutines: 100
        success_points: 1.0
        fail_penalty: 5.0
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "settings.yaml")
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	provider := NewFile(FileIn{
		Config: &FileConfig{
			Path: filePath,
		},
		Checkers: []hazycheck.Checker{},
	})

	ctx := context.Background()
	settings, err := provider.GetSettings(ctx)
	require.Error(t, err)
	require.Nil(t, settings)
}

func TestFileSettingsProvider_DifferentDurationFormats(t *testing.T) {
	yamlContent := `
services:
  - name: example
    target: http://example.com:8080
    checkers:
      - name: checker1
        run_options:
          rate:
            times: 1
            per: 1s
          max_goroutines: 10
        success_points: 1.0
        fail_penalty: 1.0
      - name: checker2
        run_options:
          rate:
            times: 1
            per: 1m
          max_goroutines: 10
        success_points: 1.0
        fail_penalty: 1.0
      - name: checker3
        run_options:
          rate:
            times: 1
            per: 1h
          max_goroutines: 10
        success_points: 1.0
        fail_penalty: 1.0
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "settings.yaml")
	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	provider := NewFile(FileIn{
		Config: &FileConfig{
			Path: filePath,
		},
		Checkers: []hazycheck.Checker{},
	})

	ctx := context.Background()
	settings, err := provider.GetSettings(ctx)
	require.NoError(t, err)
	require.NotNil(t, settings)

	service := settings.GetServices()[0]
	require.Len(t, service.GetCheckers(), 3)

	require.Equal(t, time.Second, service.GetCheckers()[0].GetRunOptions().GetRate().GetPer().AsDuration())
	require.Equal(t, time.Minute, service.GetCheckers()[1].GetRunOptions().GetRate().GetPer().AsDuration())
	require.Equal(t, time.Hour, service.GetCheckers()[2].GetRunOptions().GetRate().GetPer().AsDuration())
}
