//go:build e2e && unix
//go:build e2e && unix

package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
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
