//go:build !windows

package tts

import (
	"github.com/ebitengine/purego"
)

// dlopen opens a shared library on Unix systems
func dlopen(path string) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

// dlsym gets a symbol from a loaded library on Unix systems
func dlsym(handle uintptr, name string) (uintptr, error) {
	return purego.Dlsym(handle, name)
}

// dlclose closes a loaded library on Unix systems
func dlclose(handle uintptr) error {
	return purego.Dlclose(handle)
}
