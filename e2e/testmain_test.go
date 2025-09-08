//go:build e2e && unix

package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
	// Get the absolute path to the e2e directory
	e2eDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	// Set the absolute path for the binary
	binPath = e2eDir + "/gitagrip_e2e"

	// Build the test binary from the parent directory
	fmt.Println("Building test binary from main project...")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = ".." // Run from parent directory
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to build test binary: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(binPath)
	os.Exit(code)
}
