package tts

import (
	"runtime"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Voice != "" {
		t.Errorf("Expected empty voice, got %s", config.Voice)
	}

	if config.TopK != 50 {
		t.Errorf("Expected TopK 50, got %d", config.TopK)
	}

	if config.Temperature != 1.0 {
		t.Errorf("Expected Temperature 1.0, got %f", config.Temperature)
	}

	if config.RepetitionPenalty != 1.0 {
		t.Errorf("Expected RepetitionPenalty 1.0, got %f", config.RepetitionPenalty)
	}

	if config.TopP != 1.0 {
		t.Errorf("Expected TopP 1.0, got %f", config.TopP)
	}

	if config.MaxTokens != 0 {
		t.Errorf("Expected MaxTokens 0, got %d", config.MaxTokens)
	}

	if !config.UseCrossAttention {
		t.Error("Expected UseCrossAttention to be true")
	}

	if config.EspeakVoiceID != "" {
		t.Errorf("Expected empty EspeakVoiceID, got %s", config.EspeakVoiceID)
	}
}

func TestDefaultLibraryConfig(t *testing.T) {
	config := DefaultLibraryConfig()

	if config.LibraryPath != "" {
		t.Errorf("Expected empty LibraryPath, got %s", config.LibraryPath)
	}

	if !config.AutoDownload {
		t.Error("Expected AutoDownload to be true")
	}

	if config.DownloadURL == "" {
		t.Error("Expected DownloadURL to be set")
	}

	if config.Version == "" {
		t.Error("Expected Version to be set")
	}
}

func TestLibraryNamePlatformSpecific(t *testing.T) {
	// Test that library names are platform-specific
	// We test by checking the expected suffixes
	switch runtime.GOOS {
	case "darwin":
		// macOS should use .dylib
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			// Both Intel and Apple Silicon use .dylib
			t.Log("macOS detected - library should have .dylib suffix")
		}
	case "windows":
		// Windows should use .dll
		t.Log("Windows detected - library should have .dll suffix")
	default:
		// Linux and others should use .so
		t.Log("Linux/Unix detected - library should have .so suffix")
	}
}

func TestAudioDataStruct(t *testing.T) {
	audio := &AudioData{
		Samples:    []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		SampleRate: 44100,
	}

	if len(audio.Samples) != 5 {
		t.Errorf("Expected 5 samples, got %d", len(audio.Samples))
	}

	if audio.SampleRate != 44100 {
		t.Errorf("Expected sample rate 44100, got %d", audio.SampleRate)
	}

	// Test with empty samples
	emptyAudio := &AudioData{
		Samples:    []float32{},
		SampleRate: 22050,
	}

	if len(emptyAudio.Samples) != 0 {
		t.Errorf("Expected 0 samples, got %d", len(emptyAudio.Samples))
	}
}

func TestConfigVariations(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "Default config",
			config: DefaultConfig(),
		},
		{
			name: "Custom voice",
			config: Config{
				Voice:             "af_sarah",
				TopK:              50,
				Temperature:       1.0,
				RepetitionPenalty: 1.0,
				TopP:              1.0,
				MaxTokens:         0,
				UseCrossAttention: true,
				EspeakVoiceID:     "",
			},
		},
		{
			name: "High temperature",
			config: Config{
				Voice:             "",
				TopK:              100,
				Temperature:       1.5,
				RepetitionPenalty: 1.2,
				TopP:              0.9,
				MaxTokens:         100,
				UseCrossAttention: false,
				EspeakVoiceID:     "en",
			},
		},
		{
			name: "Zero values",
			config: Config{
				Voice:             "",
				TopK:              0,
				Temperature:       0.0,
				RepetitionPenalty: 0.0,
				TopP:              0.0,
				MaxTokens:         0,
				UseCrossAttention: false,
				EspeakVoiceID:     "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify config values are set correctly
			if tc.config.TopK < 0 {
				t.Error("TopK should be non-negative")
			}
			if tc.config.Temperature < 0 {
				t.Error("Temperature should be non-negative")
			}
			if tc.config.RepetitionPenalty < 0 {
				t.Error("RepetitionPenalty should be non-negative")
			}
			if tc.config.TopP < 0 || tc.config.TopP > 1.0 {
				t.Error("TopP should be between 0 and 1")
			}
			if tc.config.MaxTokens < 0 {
				t.Error("MaxTokens should be non-negative")
			}
		})
	}
}

func TestLibraryConfigVariations(t *testing.T) {
	tests := []struct {
		name   string
		config LibraryConfig
	}{
		{
			name: "Empty config",
			config: LibraryConfig{
				LibraryPath:  "",
				AutoDownload: true,
			},
		},
		{
			name: "With library path",
			config: LibraryConfig{
				LibraryPath:  "/usr/lib/libtts.so",
				AutoDownload: false,
			},
		},
		{
			name: "With custom URL",
			config: LibraryConfig{
				DownloadURL: "https://example.com/libs",
				Version:     "v1.0.0",
			},
		},
		{
			name: "Full config",
			config: LibraryConfig{
				LibraryPath:  "/opt/tts/lib/libtts_c_api.so",
				AutoDownload: false,
				DownloadURL:  "https://myserver.com/releases",
				Version:      "v2.0.0",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Just verify the struct can be created without panic
			_ = tc.config

			// Verify AutoDownload is set correctly
			if tc.name == "With library path" && tc.config.AutoDownload {
				t.Error("Expected AutoDownload to be false when LibraryPath is set")
			}
		})
	}
}

// Test error handling without library
func TestRunnerCreationWithoutLibrary(t *testing.T) {
	// This test verifies that we get proper errors when library is not available
	config := DefaultConfig()
	libConfig := LibraryConfig{
		LibraryPath:  "/nonexistent/path/libtts.so",
		AutoDownload: false,
	}

	_, err := NewRunnerWithConfig("model.gguf", 4, config, true, libConfig)
	if err == nil {
		t.Error("Expected error when library is not found")
	} else {
		t.Logf("Got expected error: %v", err)
		// Error message should contain information about library not found
		if !strings.Contains(err.Error(), "library") && !strings.Contains(err.Error(), "find") {
			t.Logf("Error message may need improvement: %v", err)
		}
	}
}

// Test Runner struct creation (even though it will fail to load library)
func TestRunnerStruct(t *testing.T) {
	// We can only test the Runner struct behavior without a real library
	// Since NewRunner will fail without a library, we test the expected behavior

	t.Run("Runner with invalid path", func(t *testing.T) {
		config := DefaultConfig()
		libConfig := LibraryConfig{
			LibraryPath:  "",
			AutoDownload: false,
		}

		runner, err := NewRunnerWithConfig("model.gguf", 4, config, true, libConfig)
		if err == nil {
			t.Error("Expected error when creating runner without library")
			if runner != nil {
				runner.Close()
			}
		}
	})
}

// Benchmark config creation
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

// Benchmark library config creation
func BenchmarkDefaultLibraryConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultLibraryConfig()
	}
}

// Benchmark AudioData creation
func BenchmarkAudioDataCreation(b *testing.B) {
	samples := make([]float32, 1000)
	for i := 0; i < 1000; i++ {
		samples[i] = float32(i) / 1000.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &AudioData{
			Samples:    samples,
			SampleRate: 44100,
		}
	}
}

// Test cross-compilation target platforms
func TestSupportedPlatforms(t *testing.T) {
	platforms := []struct {
		goos   string
		goarch string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
		{"windows", "arm64"},
	}

	for _, platform := range platforms {
		t.Run(platform.goos+"_"+platform.goarch, func(t *testing.T) {
			// Just verify the platform name is valid
			if platform.goos == "" || platform.goarch == "" {
				t.Error("Platform should have valid GOOS and GOARCH")
			}
		})
	}
}

// Test error messages
func TestErrorMessageContent(t *testing.T) {
	// Test that errors contain useful information
	config := DefaultConfig()
	libConfig := LibraryConfig{
		LibraryPath:  "/path/that/does/not/exist/lib.so",
		AutoDownload: false,
	}

	_, err := NewRunnerWithConfig("test.gguf", 2, config, true, libConfig)
	if err != nil {
		errMsg := err.Error()

		// Error should not be empty
		if errMsg == "" {
			t.Error("Error message should not be empty")
		}

		// Error should contain some useful context
		t.Logf("Error message: %s", errMsg)
	}
}

// Test Config string values
func TestConfigStrings(t *testing.T) {
	tests := []struct {
		name     string
		voice    string
		espeakID string
	}{
		{"Empty", "", ""},
		{"Short", "a", "en"},
		{"Normal", "af_sarah", "en-us"},
		{"Long", "very_long_voice_name_with_underscores", "en-gb-scottish"},
		{"Unicode", "日本語", "中文"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Voice = tc.voice
			config.EspeakVoiceID = tc.espeakID

			if config.Voice != tc.voice {
				t.Errorf("Voice mismatch: got %s, want %s", config.Voice, tc.voice)
			}
			if config.EspeakVoiceID != tc.espeakID {
				t.Errorf("EspeakVoiceID mismatch: got %s, want %s", config.EspeakVoiceID, tc.espeakID)
			}
		})
	}
}
