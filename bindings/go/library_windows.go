//go:build windows

package tts

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

// dlopen opens a DLL on Windows
func dlopen(path string) (uintptr, error) {
	// Load the DLL - windows.LoadLibrary accepts string directly
	handle, err := windows.LoadLibrary(path)
	if err != nil {
		return 0, fmt.Errorf("failed to load library: %w", err)
	}

	return uintptr(handle), nil
}

// dlsym gets a procedure address from a loaded DLL on Windows
func dlsym(handle uintptr, name string) (uintptr, error) {
	// Get the procedure address using syscall
	proc, err := syscall.GetProcAddress(syscall.Handle(handle), name)
	if err != nil {
		return 0, fmt.Errorf("failed to get procedure address for %s: %w", name, err)
	}

	return uintptr(proc), nil
}

// dlclose frees a loaded DLL on Windows
func dlclose(handle uintptr) error {
	return windows.FreeLibrary(windows.Handle(handle))
}
