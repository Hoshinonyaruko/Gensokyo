//go:build windows
// +build windows

package sys

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// WindowsRestarter implements the Restarter interface for Windows systems.
type WindowsRestarter struct{}

// NewRestarter creates a new Restarter appropriate for Windows systems.
func NewRestarter() *WindowsRestarter {
	return &WindowsRestarter{}
}

func (r *WindowsRestarter) Restart(executablePath string) error {
	// Separate the directory and the executable name
	execDir, execName := filepath.Split(executablePath)

	scriptContent := "@echo off\n" +
		"pushd " + strconv.Quote(execDir) + "\n" +
		"start \"\" " + strconv.Quote(execName) + "\n" +
		"popd\n"

	scriptName := "restart.bat"
	if err := os.WriteFile(scriptName, []byte(scriptContent), 0755); err != nil {
		return err
	}

	cmd := exec.Command("cmd.exe", "/C", scriptName)

	if err := cmd.Start(); err != nil {
		return err
	}

	// The current process can now exit
	os.Exit(0)

	return nil
}
