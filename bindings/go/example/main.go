// Example program demonstrating Go bindings for TTS.cpp
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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

	// Create runner
	runner, err := tts.NewRunner(*modelPath, *threads, config, *cpuOnly)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}
	defer runner.Close()

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

	// Generate audio
	fmt.Printf("Generating audio for: %s\n", *text)
	audio, err := runner.Generate(*text)
	if err != nil {
		log.Fatalf("Failed to generate audio: %v", err)
	}

	fmt.Printf("Generated %d samples at %d Hz\n", len(audio.Samples), audio.SampleRate)

	// Save to file
	if err := audio.SaveWAV(*output); err != nil {
		log.Fatalf("Failed to save audio: %v", err)
	}

	fmt.Printf("Audio saved to: %s\n", *output)
}
