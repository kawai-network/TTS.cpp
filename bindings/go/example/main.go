// Example program demonstrating Go bindings for TTS.cpp
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kawai-network/TTS.cpp/bindings/go"
)

func main() {
	var (
		modelPath = flag.String("model", "", "Path to GGUF model file (required)")
		text      = flag.String("text", "Hello, world!", "Text to synthesize")
		output    = flag.String("output", "output.wav", "Output WAV file path")
		voice     = flag.String("voice", "", "Voice to use (model dependent)")
		threads   = flag.Int("threads", 4, "Number of CPU threads")
		cpuOnly   = flag.Bool("cpu", false, "Use CPU only (no GPU)")
		listArchs = flag.Bool("list-archs", false, "List supported architectures")
	)
	flag.Parse()

	if *listArchs {
		fmt.Println("Supported model architectures:")
		for _, arch := range tts.SupportedArchitectures() {
			fmt.Printf("  - %s\n", arch)
		}
		return
	}

	if *modelPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Loading model: %s\n", *modelPath)

	// Create configuration
	config := tts.DefaultConfig()
	if *voice != "" {
		config.Voice = *voice
	}

	// Create runner with timing
	startTime := time.Now()
	runner, err := tts.NewRunner(*modelPath, *threads, config, *cpuOnly)
	loadDuration := time.Since(startTime)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}
	defer runner.Close()
	fmt.Printf("⏱️  Model loading time: %v\n", loadDuration)

	// Check if model supports voices
	if runner.SupportsVoices() {
		voices := runner.ListVoices()
		if len(voices) > 0 {
			fmt.Println("Available voices:")
			for _, v := range voices {
				fmt.Printf("  - %s\n", v)
			}
		}
	}

	// Generate audio with timing
	fmt.Printf("Generating audio for: %s\n", *text)
	genStart := time.Now()
	audio, err := runner.Generate(*text)
	genDuration := time.Since(genStart)
	if err != nil {
		log.Fatalf("Failed to generate audio: %v", err)
	}
	fmt.Printf("⏱️  Audio generation time: %v\n", genDuration)

	fmt.Printf("Generated %d samples at %d Hz\n", len(audio.Samples), audio.SampleRate)

	// Calculate audio duration and real-time factor
	audioDuration := float64(len(audio.Samples)) / float64(audio.SampleRate)
	rtf := genDuration.Seconds() / audioDuration
	fmt.Printf("⏱️  Audio duration: %.2f seconds\n", audioDuration)
	fmt.Printf("⏱️  Real-time factor (RTF): %.3fx (lower is better)\n", rtf)

	// Save to file with timing
	saveStart := time.Now()
	if err := audio.SaveWAV(*output); err != nil {
		log.Fatalf("Failed to save audio: %v", err)
	}
	saveDuration := time.Since(saveStart)
	fmt.Printf("⏱️  WAV save time: %v\n", saveDuration)

	fmt.Printf("Audio saved to: %s\n", *output)

	// Summary
	fmt.Printf("\n📊 === TIMING SUMMARY ===\n")
	fmt.Printf("📊 Model load:    %v\n", loadDuration)
	fmt.Printf("📊 Generation:    %v\n", genDuration)
	fmt.Printf("📊 WAV save:      %v\n", saveDuration)
	fmt.Printf("📊 Total:         %v\n", loadDuration+genDuration+saveDuration)
	fmt.Printf("📊 RTF:           %.3fx\n", rtf)
}
