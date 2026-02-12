package tts

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetLibraryName(t *testing.T) {
	tests := []struct {
		goos     string
		expected string
	}{
		{"linux", "libtts_c_api.so"},
		{"darwin", "libtts_c_api.dylib"},
		{"windows", "tts_c_api.dll"},
		{"freebsd", "libtts_c_api.so"},
	}

	for _, tc := range tests {
		t.Run(tc.goos, func(t *testing.T) {
			// Skip test if not on correct platform
			if runtime.GOOS != tc.goos {
				t.Skipf("Skipping test for %s (running on %s)", tc.goos, runtime.GOOS)
			}

			result := getLibraryName()
			if result != tc.expected {
				t.Errorf("getLibraryName() = %s, want %s", result, tc.expected)
			}
		})
	}
}

func TestFindLibraryInPath(t *testing.T) {
	// Test with non-existent path
	t.Run("non-existent library", func(t *testing.T) {
		// Save original LD_LIBRARY_PATH
		origLD := os.Getenv("LD_LIBRARY_PATH")
		origDYLD := os.Getenv("DYLD_LIBRARY_PATH")
		defer func() {
			os.Setenv("LD_LIBRARY_PATH", origLD)
			os.Setenv("DYLD_LIBRARY_PATH", origDYLD)
		}()

		// Clear library paths
		os.Setenv("LD_LIBRARY_PATH", "")
		os.Setenv("DYLD_LIBRARY_PATH", "")

		result := findLibraryInPath()
		if result != "" {
			t.Logf("Library found in path: %s", result)
		}
		// Test passes whether library is found or not
	})
}

func TestTryExtractOrDownload_CacheDir(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	config := LibraryConfig{
		Version:      "v0.1.1",
		DownloadURL:  "https://github.com/kawai-network/TTS.cpp/releases/download",
		AutoDownload: false, // Don't actually download in unit test
	}

	// Test with non-existent library (should fail gracefully)
	t.Run("library not found without download", func(t *testing.T) {
		// Override cache dir temporarily
		origCache := os.Getenv("XDG_CACHE_HOME")
		os.Setenv("XDG_CACHE_HOME", tempDir)
		defer os.Setenv("XDG_CACHE_HOME", origCache)

		result := tryExtractOrDownload(config)
		if result != "" {
			t.Errorf("Expected empty result when library not found and download disabled, got: %s", result)
		}
	})
}

func TestDownloadLibrary_WithMock(t *testing.T) {
	// This test verifies download logic without actually downloading
	t.Run("invalid download URL", func(t *testing.T) {
		config := LibraryConfig{
			Version:      "v0.1.1",
			DownloadURL:  "", // Empty URL should fail
			AutoDownload: true,
		}

		tempDir := t.TempDir()
		result := downloadLibrary(config, tempDir)

		if result {
			t.Error("Expected download to fail with empty URL")
		}
	})

	t.Run("unsupported platform", func(t *testing.T) {
		// Save original GOOS
		origGOOS := runtime.GOOS

		config := LibraryConfig{
			Version:      "v0.1.1",
			DownloadURL:  "https://example.com",
			AutoDownload: true,
		}

		tempDir := t.TempDir()

		// This test will only run on unsupported platforms
		// On supported platforms (linux, darwin, windows), we can't test this
		if origGOOS == "linux" || origGOOS == "darwin" || origGOOS == "windows" {
			t.Skipf("Skipping test on supported platform: %s", origGOOS)
		}

		result := downloadLibrary(config, tempDir)
		if result {
			t.Error("Expected download to fail on unsupported platform")
		}
	})
}

func TestExtractZip_InvalidData(t *testing.T) {
	t.Run("invalid zip data", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create invalid zip data
		invalidData := []byte("not a zip file")

		result := extractZip(
			&mockReaderAt{data: invalidData},
			int64(len(invalidData)),
			tempDir,
		)

		if result {
			t.Error("Expected extractZip to fail with invalid data")
		}
	})
}

// mockReaderAt implements io.ReaderAt for testing
type mockReaderAt struct {
	data []byte
}

func (m *mockReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(m.data)) {
		return 0, nil
	}
	n = copy(p, m.data[off:])
	if n < len(p) {
		return n, nil
	}
	return n, nil
}

