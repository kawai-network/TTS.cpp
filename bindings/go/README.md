# Go Bindings for TTS.cpp (CGO-Free)

This package provides **CGO-free** Go bindings for the TTS.cpp text-to-speech library using [purego](https://github.com/ebitengine/purego).

## Features

- ✅ **No CGO required** - Pure Go implementation
- ✅ **Easy cross-compilation** - Just set GOOS/GOARCH
- ✅ **Auto-download libraries** - Automatically downloads shared libraries from GitHub releases
- ✅ **Simple distribution** - `go get` and use immediately
- ✅ **Multiple platforms** - Linux, macOS, Windows support

## Installation

```bash
go get github.com/kawai-network/TTS.cpp/bindings/go
```

**Requirements:**
- Go 1.21 or later
- No C compiler required!

## Quick Start

```go
package main

import (
    "log"
    "github.com/kawai-network/TTS.cpp/bindings/go/tts"
)

func main() {
    // Configuration
    config := tts.DefaultConfig()
    config.Voice = "af_sarah"  // For Kokoro models
    
    // Create runner (auto-downloads library if needed)
    runner, err := tts.NewRunner(
        "path/to/model.gguf",
        4,      // number of threads
        config,
        false,  // use GPU if available
    )
    if err != nil {
        log.Fatal(err)
    }
    defer runner.Close()
    
    // Generate audio
    audio, err := runner.Generate("Hello, this is a test.")
    if err != nil {
        log.Fatal(err)
    }
    
    // Save to file
    err = audio.SaveWAV("output.wav")
    if err != nil {
        log.Fatal(err)
    }
}
```

## Library Configuration

By default, the library will auto-download from GitHub releases. You can customize this:

```go
libConfig := tts.LibraryConfig{
    LibraryPath:  "/path/to/libtts_c_api.so",  // Use local library
    AutoDownload: false,                         // Disable auto-download
}

runner, err := tts.NewRunnerWithConfig(
    "model.gguf", 
    4, 
    config, 
    false, 
    libConfig,
)
```

### Library Loading Order

1. If `LibraryPath` is set and exists, use it
2. Search in common system paths
3. Search in same directory as executable
4. Check cache directory (`~/.cache/tts-go/`)
5. If `AutoDownload` is true, download from GitHub releases

### Environment Variables

- `LD_LIBRARY_PATH` (Linux) - Additional library search paths
- `DYLD_LIBRARY_PATH` (macOS) - Additional library search paths
- `PATH` (Windows) - DLL search paths

## Cross-Compilation

Since there's no CGO, cross-compilation is simple:

```bash
# Build for Linux AMD64
GOOS=linux GOARCH=amd64 go build

# Build for macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build

# Build for Windows
GOOS=windows GOARCH=amd64 go build
```

**Note:** The shared library will be downloaded at runtime on the target machine.

## API Reference

### Types

- `tts.Runner`: TTS model runner
- `tts.Config`: Configuration for generation
- `tts.AudioData`: Generated audio data
- `tts.LibraryConfig`: Library loading configuration

### Functions

- `tts.DefaultConfig()`: Returns default configuration
- `tts.DefaultLibraryConfig()`: Returns default library configuration
- `tts.NewRunner(modelPath, nThreads, config, cpuOnly)`: Creates a new runner
- `tts.NewRunnerWithConfig(...)`: Creates runner with library config
- `tts.SupportedArchitectures()`: Returns list of supported model architectures

### Runner Methods

- `Generate(text string) (*AudioData, error)`: Generate audio from text
- `ListVoices() []string`: Get available voices
- `SupportsVoices() bool`: Check if model supports voice selection
- `UpdateConditionalPrompt(encoderPath, prompt string) error`: Update conditional prompt
- `Close()`: Free resources

### AudioData Methods

- `SaveWAV(path string) error`: Save audio to WAV file

## Supported Models

- Parler TTS (Mini & Large v1)
- Kokoro
- Dia
- Orpheus

## Platform Support

| Platform | Status | Library Names |
|----------|--------|---------------|
| Linux AMD64 | ✅ Supported | `libtts_c_api.so` |
| macOS Intel | ✅ Supported | `libtts_c_api.dylib` |
| macOS ARM64 | ✅ Supported | `libtts_c_api.dylib` |
| Windows AMD64 | ✅ Supported | `tts_c_api.dll` |

## How It Works

1. **Runtime Library Loading**: Uses `purego.Dlopen()` to load shared library at runtime
2. **Symbol Resolution**: Dynamically resolves C function symbols
3. **FFI Calls**: Uses system FFI to call C functions from Go
4. **Auto-Download**: Downloads pre-built libraries from GitHub releases
5. **Memory Management**: Careful handling of C-allocated memory

## Building the C API Library

If you want to build the C API library yourself:

```bash
# Clone and build TTS.cpp
git clone --recursive https://github.com/kawai-network/TTS.cpp.git
cd TTS.cpp

# Build shared library with C API
cmake -B build-shared \
    -DBUILD_SHARED_LIBS=ON \
    -DTTS_BUILD_EXAMPLES=OFF \
    -DTTS_BUILD_C_API=ON
cmake --build build-shared --config Release

# The library will be at:
# - build-shared/bindings/go/libtts_c_api.so (Linux)
# - build-shared/bindings/go/libtts_c_api.dylib (macOS)
# - build-shared/bindings/go/tts_c_api.dll (Windows)
```

## Troubleshooting

### Library Not Found

If you get "could not find TTS library":

1. Set `LibraryPath` to the absolute path of the library
2. Or enable `AutoDownload: true` (default)
3. Or place the library in the same directory as your executable

### Download Fails

If auto-download fails:

1. Download manually from [GitHub Releases](https://github.com/kawai-network/TTS.cpp/releases)
2. Extract to `~/.cache/tts-go/v0.1.0/` (Linux/macOS) or appropriate location
3. Or set `LibraryPath` to the downloaded library

### Version Mismatch

Ensure the library version matches the Go binding version:

```go
libConfig := tts.LibraryConfig{
    Version: "v0.1.0",  // Match the release version
}
```

## Performance Notes

- First call to `NewRunner()` may take time due to library download
- Audio generation performance is identical to C++ version
- Memory copying between Go and C adds minimal overhead
- Consider reusing `Runner` instances for multiple generations

## License

MIT License - same as TTS.cpp

## Contributing

Contributions welcome! Please ensure:
- Code works with `CGO_ENABLED=0`
- Tested on multiple platforms
- Documentation is updated
# Trigger workflow
