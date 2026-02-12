// Package tts provides CGO-free Go bindings for TTS.cpp using purego
package tts

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Embedded libraries will be available here when bundled
// For now, libraries are auto-downloaded from GitHub releases
// var embeddedLibs embed.FS

// LibraryConfig holds configuration for library loading
type LibraryConfig struct {
	LibraryPath  string // Path to shared library (if empty, will try to auto-download)
	AutoDownload bool   // Whether to auto-download library if not found
	DownloadURL  string // Base URL for downloading library
	Version      string // Library version to download
}

// DefaultLibraryConfig returns default library configuration
func DefaultLibraryConfig() LibraryConfig {
	return LibraryConfig{
		LibraryPath:  "",
		AutoDownload: true,
		DownloadURL:  "https://github.com/kawai-network/TTS.cpp/releases/download",
		Version:      "v0.1.0",
	}
}

// runnerLibs holds the loaded library functions
type runnerLibs struct {
	libHandle uintptr

	// Function pointers (stored as uintptr for SyscallN)
	configDefault             uintptr
	runnerCreate              uintptr
	generate                  uintptr
	audioDataFree             uintptr
	runnerFree                uintptr
	getError                  uintptr
	listVoices                uintptr
	freeVoices                uintptr
	updateConditionalPrompt   uintptr
	supportsVoices            uintptr
	getSupportedArchitectures uintptr
	freeArchitectures         uintptr
	saveAudioWav              uintptr
}

var (
	libs     *runnerLibs
	libsOnce sync.Once
	libsErr  error
)

// loadLibrary loads the shared library and resolves symbols
func loadLibrary(config LibraryConfig) (*runnerLibs, error) {
	var libPath string

	// Try provided path first
	if config.LibraryPath != "" {
		if _, err := os.Stat(config.LibraryPath); err == nil {
			libPath = config.LibraryPath
		}
	}

	// Try to find in common locations
	if libPath == "" {
		libPath = findLibraryInPath()
	}

	// Try to extract from embedded or download
	if libPath == "" && config.AutoDownload {
		libPath = tryExtractOrDownload(config)
	}

	if libPath == "" {
		return nil, fmt.Errorf("could not find TTS library. Please set LibraryPath or enable AutoDownload")
	}

	// Load the library
	handle, err := purego.Dlopen(libPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, fmt.Errorf("failed to load library %s: %w", libPath, err)
	}

	rl := &runnerLibs{libHandle: handle}

	// Resolve function symbols using Dlsym
	rl.configDefault, err = purego.Dlsym(handle, "tts_config_default")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_config_default: %w", err)
	}
	rl.runnerCreate, err = purego.Dlsym(handle, "tts_runner_create")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_runner_create: %w", err)
	}
	rl.generate, err = purego.Dlsym(handle, "tts_generate")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_generate: %w", err)
	}
	rl.audioDataFree, err = purego.Dlsym(handle, "tts_audio_data_free")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_audio_data_free: %w", err)
	}
	rl.runnerFree, err = purego.Dlsym(handle, "tts_runner_free")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_runner_free: %w", err)
	}
	rl.getError, err = purego.Dlsym(handle, "tts_get_error")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_get_error: %w", err)
	}
	rl.listVoices, err = purego.Dlsym(handle, "tts_list_voices")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_list_voices: %w", err)
	}
	rl.freeVoices, err = purego.Dlsym(handle, "tts_free_voices")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_free_voices: %w", err)
	}
	rl.updateConditionalPrompt, err = purego.Dlsym(handle, "tts_update_conditional_prompt")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_update_conditional_prompt: %w", err)
	}
	rl.supportsVoices, err = purego.Dlsym(handle, "tts_supports_voices")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_supports_voices: %w", err)
	}
	rl.getSupportedArchitectures, err = purego.Dlsym(handle, "tts_get_supported_architectures")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_get_supported_architectures: %w", err)
	}
	rl.freeArchitectures, err = purego.Dlsym(handle, "tts_free_architectures")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_free_architectures: %w", err)
	}
	rl.saveAudioWav, err = purego.Dlsym(handle, "tts_save_audio_wav")
	if err != nil {
		return nil, fmt.Errorf("failed to find tts_save_audio_wav: %w", err)
	}

	return rl, nil
}

// getLibraryName returns the platform-specific library name
func getLibraryName() string {
	switch runtime.GOOS {
	case "darwin":
		return "libtts_c_api.dylib"
	case "windows":
		return "tts_c_api.dll"
	default: // linux and others
		return "libtts_c_api.so"
	}
}

// findLibraryInPath searches for library in common locations
func findLibraryInPath() string {
	libName := getLibraryName()

	// Check in same directory as binary
	if execPath, err := os.Executable(); err == nil {
		dir := filepath.Dir(execPath)
		path := filepath.Join(dir, libName)
		if _, err := os.Stat(path); err == nil {
			return path
		}
		// Check in lib subdirectory
		path = filepath.Join(dir, "lib", libName)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Check in system paths
	switch runtime.GOOS {
	case "darwin":
		paths := []string{
			"/usr/local/lib/" + libName,
			"/opt/homebrew/lib/" + libName,
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	case "linux":
		paths := []string{
			"/usr/lib/" + libName,
			"/usr/local/lib/" + libName,
			"/opt/tts/lib/" + libName,
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	// Check LD_LIBRARY_PATH or DYLD_LIBRARY_PATH
	libEnv := os.Getenv("LD_LIBRARY_PATH")
	if runtime.GOOS == "darwin" {
		libEnv = os.Getenv("DYLD_LIBRARY_PATH")
	}

	for _, dir := range strings.Split(libEnv, ":") {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, libName)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// tryExtractOrDownload attempts to extract embedded library or download it
func tryExtractOrDownload(config LibraryConfig) string {
	libName := getLibraryName()
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = "/tmp"
	}

	ttsDir := filepath.Join(cacheDir, "tts-go", config.Version)
	libPath := filepath.Join(ttsDir, libName)

	// Check if already cached
	if _, err := os.Stat(libPath); err == nil {
		return libPath
	}

	// Try to extract from embedded first
	if extractEmbeddedLibrary(ttsDir, libName) {
		return libPath
	}

	// Try to download
	if config.DownloadURL != "" {
		if downloadLibrary(config, ttsDir) {
			return libPath
		}
	}

	return ""
}

// extractEmbeddedLibrary extracts library from embedded FS
// Note: For now, libraries are downloaded from GitHub releases instead
func extractEmbeddedLibrary(destDir, libName string) bool {
	// TODO: Implement embedded library extraction when libraries are bundled
	// For now, return false to trigger download from GitHub releases
	return false
}

// extractZip extracts a zip file to destination directory
func extractZip(r io.ReaderAt, size int64, destDir string) bool {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return false
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return false
	}

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}

		path := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			rc.Close()
			continue
		}

		os.MkdirAll(filepath.Dir(path), 0755)
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			continue
		}

		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}

	return true
}

// downloadLibrary downloads library from GitHub releases
func downloadLibrary(config LibraryConfig, destDir string) bool {
	platform := runtime.GOOS + "-" + runtime.GOARCH
	zipURL := fmt.Sprintf("%s/%s/tts-shared-%s.zip", config.DownloadURL, config.Version, platform)

	// Determine which compiler variant to use
	if runtime.GOOS == "linux" {
		// Try gcc first, then clang
		zipURL = fmt.Sprintf("%s/%s/tts-shared-linux-gcc.zip", config.DownloadURL, config.Version)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", zipURL, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	return extractZip(bytes.NewReader(data), int64(len(data)), destDir)
}

// ensureLibraryInitialized ensures library is loaded
func ensureLibraryInitialized(config LibraryConfig) (*runnerLibs, error) {
	libsOnce.Do(func() {
		libs, libsErr = loadLibrary(config)
	})
	return libs, libsErr
}

// Config represents TTS generation configuration
type Config struct {
	Voice             string
	TopK              int
	Temperature       float32
	RepetitionPenalty float32
	TopP              float32
	MaxTokens         int
	UseCrossAttention bool
	EspeakVoiceID     string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Voice:             "",
		TopK:              50,
		Temperature:       1.0,
		RepetitionPenalty: 1.0,
		TopP:              1.0,
		MaxTokens:         0,
		UseCrossAttention: true,
		EspeakVoiceID:     "",
	}
}

// cConfig mirrors the C tts_config structure
// Must match the C struct layout exactly!
type cConfig struct {
	voice             *byte
	topK              int32
	temperature       float32
	repetitionPenalty float32
	topP              float32
	maxTokens         int32
	useCrossAttention bool
	espeakVoiceID     *byte
}

// cAudioData mirrors the C tts_audio_data structure
type cAudioData struct {
	data       *float32
	nOutputs   uintptr
	sampleRate uint32
}

// AudioData represents generated audio
type AudioData struct {
	Samples    []float32
	SampleRate uint32
}

// Runner represents a TTS model runner
type Runner struct {
	handle uintptr
	libs   *runnerLibs
	config Config
}

// NewRunner creates a new TTS runner from a model file
func NewRunner(modelPath string, nThreads int, config Config, cpuOnly bool) (*Runner, error) {
	return NewRunnerWithConfig(modelPath, nThreads, config, cpuOnly, DefaultLibraryConfig())
}

// NewRunnerWithConfig creates a new runner with library configuration
func NewRunnerWithConfig(modelPath string, nThreads int, config Config, cpuOnly bool, libConfig LibraryConfig) (*Runner, error) {
	l, err := ensureLibraryInitialized(libConfig)
	if err != nil {
		return nil, err
	}

	// Prepare C config
	cCfg := cConfig{
		topK:              int32(config.TopK),
		temperature:       config.Temperature,
		repetitionPenalty: config.RepetitionPenalty,
		topP:              config.TopP,
		maxTokens:         int32(config.MaxTokens),
		useCrossAttention: config.UseCrossAttention,
	}

	// Handle strings
	if config.Voice != "" {
		voiceBytes := append([]byte(config.Voice), 0)
		cCfg.voice = &voiceBytes[0]
	}
	if config.EspeakVoiceID != "" {
		espeakBytes := append([]byte(config.EspeakVoiceID), 0)
		cCfg.espeakVoiceID = &espeakBytes[0]
	}

	// Create runner
	modelBytes := append([]byte(modelPath), 0)
	handle, _, _ := purego.SyscallN(
		l.runnerCreate,
		uintptr(unsafe.Pointer(&modelBytes[0])),
		uintptr(nThreads),
		uintptr(unsafe.Pointer(&cCfg)),
		uintptr(boolToInt(cpuOnly)),
	)

	if handle == 0 {
		errStr := getErrorString(l)
		return nil, fmt.Errorf("failed to create runner: %s", errStr)
	}

	return &Runner{
		handle: handle,
		libs:   l,
		config: config,
	}, nil
}

// getErrorString gets the last error message
func getErrorString(l *runnerLibs) string {
	ptr, _, _ := purego.SyscallN(l.getError)
	if ptr == 0 {
		return "unknown error"
	}
	return goString(ptr)
}

// goString converts C string to Go string
func goString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}

	// Find null terminator
	var length int
	for {
		b := *(*byte)(unsafe.Pointer(ptr + uintptr(length)))
		if b == 0 {
			break
		}
		length++
	}

	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length))
}

// boolToInt converts bool to int (0 or 1)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Generate generates audio from text
func (r *Runner) Generate(text string) (*AudioData, error) {
	textBytes := append([]byte(text), 0)

	var output cAudioData
	ret, _, _ := purego.SyscallN(
		r.libs.generate,
		r.handle,
		uintptr(unsafe.Pointer(&textBytes[0])),
		uintptr(unsafe.Pointer(&output)),
	)

	if ret == 0 {
		errStr := getErrorString(r.libs)
		return nil, fmt.Errorf("generation failed: %s", errStr)
	}

	// Copy data to Go slice
	nOutputs := int(output.nOutputs)
	samples := make([]float32, nOutputs)
	if nOutputs > 0 && output.data != nil {
		cSlice := unsafe.Slice(output.data, nOutputs)
		copy(samples, cSlice)
	}

	// Free C memory
	purego.SyscallN(r.libs.audioDataFree, uintptr(unsafe.Pointer(&output)))

	return &AudioData{
		Samples:    samples,
		SampleRate: output.sampleRate,
	}, nil
}

// ListVoices returns list of available voices
func (r *Runner) ListVoices() []string {
	ptr, _, _ := purego.SyscallN(r.libs.listVoices, r.handle)
	if ptr == 0 {
		return nil
	}
	defer purego.SyscallN(r.libs.freeVoices, ptr)

	var voices []string
	for {
		voicePtr := *(*uintptr)(unsafe.Pointer(ptr))
		if voicePtr == 0 {
			break
		}
		voices = append(voices, goString(voicePtr))
		ptr += unsafe.Sizeof(uintptr(0))
	}

	return voices
}

// SupportsVoices returns true if the model supports voice selection
func (r *Runner) SupportsVoices() bool {
	ret, _, _ := purego.SyscallN(r.libs.supportsVoices, r.handle)
	return ret != 0
}

// UpdateConditionalPrompt updates the conditional prompt
func (r *Runner) UpdateConditionalPrompt(textEncoderPath, prompt string) error {
	encoderBytes := append([]byte(textEncoderPath), 0)
	promptBytes := append([]byte(prompt), 0)

	ret, _, _ := purego.SyscallN(
		r.libs.updateConditionalPrompt,
		r.handle,
		uintptr(unsafe.Pointer(&encoderBytes[0])),
		uintptr(unsafe.Pointer(&promptBytes[0])),
	)

	if ret == 0 {
		return fmt.Errorf("failed to update conditional prompt: %s", getErrorString(r.libs))
	}
	return nil
}

// SaveWAV saves audio data to a WAV file
func (a *AudioData) SaveWAV(filePath string) error {
	l, err := ensureLibraryInitialized(DefaultLibraryConfig())
	if err != nil {
		return err
	}

	// Prepare audio data struct
	cData := cAudioData{
		nOutputs:   uintptr(len(a.Samples)),
		sampleRate: a.SampleRate,
	}

	if len(a.Samples) > 0 {
		cData.data = &a.Samples[0]
	}

	pathBytes := append([]byte(filePath), 0)
	ret, _, _ := purego.SyscallN(
		l.saveAudioWav,
		uintptr(unsafe.Pointer(&cData)),
		uintptr(unsafe.Pointer(&pathBytes[0])),
		uintptr(unsafe.Pointer(&a.SampleRate)),
	)

	if ret == 0 {
		return fmt.Errorf("failed to save audio: %s", getErrorString(l))
	}
	return nil
}

// Close frees the runner resources
func (r *Runner) Close() {
	if r.handle != 0 && r.libs != nil {
		purego.SyscallN(r.libs.runnerFree, r.handle)
		r.handle = 0
	}
}

// SupportedArchitectures returns list of supported model architectures
func SupportedArchitectures() []string {
	l, err := ensureLibraryInitialized(DefaultLibraryConfig())
	if err != nil {
		return nil
	}

	ptr, _, _ := purego.SyscallN(l.getSupportedArchitectures)
	if ptr == 0 {
		return nil
	}
	defer purego.SyscallN(l.freeArchitectures, ptr)

	var archs []string
	for {
		archPtr := *(*uintptr)(unsafe.Pointer(ptr))
		if archPtr == 0 {
			break
		}
		archs = append(archs, goString(archPtr))
		ptr += unsafe.Sizeof(uintptr(0))
	}

	return archs
}
