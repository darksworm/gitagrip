#!/bin/bash

echo "Testing GitaGrip with new hexagonal architecture..."
echo "=================================================="
echo ""

# Set environment variable to use new architecture
export USE_NEW_ARCHITECTURE=1

# Enable debug logging
export RUST_LOG=gitagrip=debug,gitagrip_core=debug

# Run the app
cargo run --quiet

echo ""
echo "Test complete. If the app started without errors, the new architecture is working!"