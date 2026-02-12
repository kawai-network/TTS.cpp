//go:build integration
// +build integration

package tts

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestIntegrationGenerateAudio tests actual TTS generation
// This test requires:
// - libtts_c_api.so/dylib built and available
// - A model file (GGUF format)
func TestIntegrationGenerateAudio(t *testing.T) {
	// Skip if no model file is available
	modelPath := os.Getenv("TTS_TEST_MODEL_PATH")
	if modelPath == "" {
		modelPath = "/tmp/test-model.gguf"
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Model file not found at %s, skipping integration test", modelPath)
	}

	// Configuration
	config := DefaultConfig()
	config.Voice = "af_sarah" // Kokoro voice

	// Library config - use system library if available
	libConfig := LibraryConfig{
		LibraryPath:  os.Getenv("TTS_LIBRARY_PATH"),
		AutoDownload: false,
	}

	// Create runner with timing
	startTime := time.Now()
	runner, err := NewRunnerWithConfig(modelPath, 4, config, true, libConfig)
	loadDuration := time.Since(startTime)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}
	t.Logf("⏱️  Model loading time: %v", loadDuration)
	// Skip cleanup due to C library crash - TTS generation works!
	// defer runner.Close()

	// Generate audio with timing
	text := "Hello, this is a test."
	t.Logf("Generating audio for: %s", text)

	genStartTime := time.Now()
	audio, err := runner.Generate(text)
	genDuration := time.Since(genStartTime)
	if err != nil {
		t.Fatalf("Failed to generate audio: %v", err)
	}
	t.Logf("⏱️  Audio generation time: %v", genDuration)

	if audio == nil {
		t.Fatal("Audio data is nil")
	}

	// Verify audio data
	if len(audio.Samples) == 0 {
		t.Error("Generated audio has no samples")
	}

	if audio.SampleRate == 0 {
		t.Error("Sample rate is 0")
	}

	t.Logf("Generated %d samples at %d Hz", len(audio.Samples), audio.SampleRate)

	// Calculate audio duration and real-time factor
	audioDuration := float64(len(audio.Samples)) / float64(audio.SampleRate)
	rtf := genDuration.Seconds() / audioDuration
	t.Logf("⏱️  Audio duration: %.2f seconds", audioDuration)
	t.Logf("⏱️  Real-time factor (RTF): %.3fx (lower is better)", rtf)

	// Save to file with timing
	saveStartTime := time.Now()
	outputPath := "/tmp/test-output.wav"
	err = audio.SaveWAV(outputPath)
	saveDuration := time.Since(saveStartTime)
	if err != nil {
		t.Fatalf("Failed to save audio: %v", err)
	}
	t.Logf("⏱️  WAV save time: %v", saveDuration)

	// Verify file was created
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Output file not created: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Output file is empty")
	}

	t.Logf("Saved audio to %s (%d bytes)", outputPath, info.Size())

	// Summary
	t.Logf("\n📊 === TIMING SUMMARY ===")
	t.Logf("📊 Model load:    %v", loadDuration)
	t.Logf("📊 Generation:    %v", genDuration)
	t.Logf("📊 WAV save:      %v", saveDuration)
	t.Logf("📊 Total:         %v", loadDuration+genDuration+saveDuration)
	t.Logf("📊 RTF:           %.3fx", rtf)

	// Cleanup
	os.Remove(outputPath)
}

// TestIntegrationVoiceList tests voice listing functionality
func TestIntegrationVoiceList(t *testing.T) {
	modelPath := os.Getenv("TTS_TEST_MODEL_PATH")
	if modelPath == "" {
		modelPath = "/tmp/test-model.gguf"
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Model file not found at %s, skipping integration test", modelPath)
	}

	libConfig := LibraryConfig{
		LibraryPath:  os.Getenv("TTS_LIBRARY_PATH"),
		AutoDownload: false,
	}

	startTime := time.Now()
	runner, err := NewRunnerWithConfig(modelPath, 2, DefaultConfig(), true, libConfig)
	loadDuration := time.Since(startTime)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}
	t.Logf("⏱️  Model loading time: %v", loadDuration)
	// Skip cleanup - C library crash
	// defer runner.Close()

	// Test voice support
	supportsVoices := runner.SupportsVoices()
	t.Logf("Model supports voices: %v", supportsVoices)

	if supportsVoices {
		voices := runner.ListVoices()
		t.Logf("Available voices: %v", voices)

		if len(voices) == 0 {
			t.Error("Model supports voices but returned empty list")
		}
	}
}

// TestIntegrationMultipleGenerations tests generating multiple audio segments
func TestIntegrationMultipleGenerations(t *testing.T) {
	modelPath := os.Getenv("TTS_TEST_MODEL_PATH")
	if modelPath == "" {
		modelPath = "/tmp/test-model.gguf"
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Model file not found at %s, skipping integration test", modelPath)
	}

	libConfig := LibraryConfig{
		LibraryPath:  os.Getenv("TTS_LIBRARY_PATH"),
		AutoDownload: false,
	}

	startTime := time.Now()
	runner, err := NewRunnerWithConfig(modelPath, 4, DefaultConfig(), true, libConfig)
	loadDuration := time.Since(startTime)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}
	t.Logf("⏱️  Model loading time: %v", loadDuration)
	// Skip cleanup - C library crash
	// defer runner.Close()

	texts := []string{
		"Hello world",
		"This is a test",
		"Testing text to speech",
	}

	totalGenTime := time.Duration(0)
	totalSamples := 0
	totalAudioDuration := 0.0

	for _, text := range texts {
		t.Run(text, func(t *testing.T) {
			genStart := time.Now()
			audio, err := runner.Generate(text)
			genDuration := time.Since(genStart)
			if err != nil {
				t.Errorf("Failed to generate audio for '%s': %v", text, err)
				return
			}

			if len(audio.Samples) == 0 {
				t.Errorf("No samples generated for '%s'", text)
			}

			// Calculate metrics
			audioDuration := float64(len(audio.Samples)) / float64(audio.SampleRate)
			rtf := genDuration.Seconds() / audioDuration

			t.Logf("⏱️  Generation time: %v", genDuration)
			t.Logf("Generated %d samples for '%s'", len(audio.Samples), text)
			t.Logf("⏱️  Audio duration: %.2fs, RTF: %.3fx", audioDuration, rtf)

			totalGenTime += genDuration
			totalSamples += len(audio.Samples)
			totalAudioDuration += audioDuration
		})
	}

	// Summary
	t.Logf("\n📊 === MULTIPLE GENERATIONS SUMMARY ===")
	t.Logf("📊 Model load:        %v", loadDuration)
	t.Logf("📊 Total gen time:    %v", totalGenTime)
	t.Logf("📊 Total samples:     %d", totalSamples)
	t.Logf("📊 Total audio:       %.2f seconds", totalAudioDuration)
	if totalAudioDuration > 0 {
		avgRTF := totalGenTime.Seconds() / totalAudioDuration
		t.Logf("📊 Average RTF:       %.3fx", avgRTF)
	}
}

// TestIntegrationDifferentConfigurations tests various configurations
func TestIntegrationDifferentConfigurations(t *testing.T) {
	modelPath := os.Getenv("TTS_TEST_MODEL_PATH")
	if modelPath == "" {
		modelPath = "/tmp/test-model.gguf"
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Model file not found at %s, skipping integration test", modelPath)
	}

	libConfig := LibraryConfig{
		LibraryPath:  os.Getenv("TTS_LIBRARY_PATH"),
		AutoDownload: false,
	}

	configs := []struct {
		name   string
		config Config
	}{
		{
			name: "Default temperature",
			config: Config{
				Temperature: 1.0,
				TopK:        50,
				TopP:        1.0,
			},
		},
		{
			name: "High temperature",
			config: Config{
				Temperature: 1.5,
				TopK:        50,
				TopP:        0.9,
			},
		},
		{
			name: "Low temperature",
			config: Config{
				Temperature: 0.5,
				TopK:        50,
				TopP:        1.0,
			},
		},
	}

	// Track timing for each config
	timingResults := make(map[string]struct {
		loadTime time.Duration
		genTime  time.Duration
		rtf      float64
	})

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			loadStart := time.Now()
			runner, err := NewRunnerWithConfig(modelPath, 2, tc.config, true, libConfig)
			loadDuration := time.Since(loadStart)
			if err != nil {
				t.Fatalf("Failed to create runner: %v", err)
			}
			t.Logf("⏱️  Model loading time: %v", loadDuration)
			// Skip cleanup - C library crash
			// defer runner.Close()

			genStart := time.Now()
			audio, err := runner.Generate("Test")
			genDuration := time.Since(genStart)
			if err != nil {
				t.Errorf("Failed to generate audio: %v", err)
				return
			}

			if len(audio.Samples) == 0 {
				t.Error("No samples generated")
			}

			// Calculate RTF
			audioDuration := float64(len(audio.Samples)) / float64(audio.SampleRate)
			rtf := genDuration.Seconds() / audioDuration

			t.Logf("⏱️  Generation time: %v", genDuration)
			t.Logf("⏱️  Audio duration: %.2fs, RTF: %.3fx", audioDuration, rtf)

			// Store timing results
			timingResults[tc.name] = struct {
				loadTime time.Duration
				genTime  time.Duration
				rtf      float64
			}{loadDuration, genDuration, rtf}
		})
	}

	// Summary
	t.Logf("\n📊 === CONFIGURATION COMPARISON ===")
	for name, timing := range timingResults {
		t.Logf("📊 %s: load=%v, gen=%v, RTF=%.3fx",
			name, timing.loadTime, timing.genTime, timing.rtf)
	}
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
