#!/bin/bash

echo "🚀 GitaGrip Go TUI Test Framework"
echo "================================="
echo ""

# Check if we're in the right directory
if [[ ! -f "./tui_test.go" ]]; then
    echo "❌ Please run this script from the e2e directory"
    echo "   cd e2e && ./run_tests.sh"
    echo ""
    exit 1
fi

# Ensure dependencies are ready
echo "🔧 Ensuring Go dependencies..."
go mod tidy
echo ""

# Run Go tests with verbose output
echo "🧪 Running Go TUI tests..."
echo ""

# Run tests with timeout and verbose output
go test -v -timeout=60s ./...

echo ""
echo "✅ Test run completed!"
echo ""
echo "💡 Individual test commands:"
echo "   go test -v -run TestHelpCommand"
echo "   go test -v -run TestBasicRepositoryDiscovery"
echo "   go test -v -run TestKeyboardNavigation"
echo "   go test -v -run TestRepositorySelection"
echo "   go test -v -run TestConfigFileCreation"
echo "   go test -v -run TestApplicationExit"
echo ""
echo "🔍 For detailed output:"
echo "   go test -v -timeout=60s | tee test_output.log"