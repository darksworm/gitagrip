#!/bin/bash

# Stress test script for E2E tests
# Runs tests multiple times and collects statistics

set -e

# Run from e2e directory
cd "$(dirname "$0")"
TEST_BINARY="./e2e.test"
TEST_CMD="$TEST_BINARY -test.v -test.timeout=30s"
RUNS=${1:-100}  # Default to 100 runs if no argument provided
PARALLEL=${2:-10}  # Default to 10 parallel runs
LOG_FILE="e2e/stress_test_results.log"

# Build test binary once at the beginning
echo "Building test binary..."
if [ ! -f "$TEST_BINARY" ]; then
    go test -c -o "$TEST_BINARY"
    echo "Test binary built successfully"
else
    echo "Test binary already exists, skipping build"
fi

echo "Starting E2E stress test with $RUNS runs ($PARALLEL parallel)..."
echo "Results will be logged to $LOG_FILE"
echo "Test command: $TEST_CMD"
echo

# Clean up any existing log
rm -f "$LOG_FILE"

# Initialize counters
total_runs=0
passed_runs=0
failed_runs=0
declare -a runtimes

start_time=$(date +%s)

# Function to run a single test
run_test() {
    local run_id=$1
    local temp_log="/tmp/test_output_$run_id.log"

    # Time the test run
    run_start=$(date +%s.%3N)

    if go test -v -timeout=30s > "$temp_log" 2>&1; then
        run_end=$(date +%s.%3N)
        runtime=$(echo "$run_end - $run_start" | bc -l)
        status="PASS"
        echo "$run_id:$status:$runtime" >&3
    else
        run_end=$(date +%s.%3N)
        runtime=$(echo "$run_end - $run_start" | bc -l)
        status="FAIL"
        echo "$run_id:$status:$runtime" >&3
        echo "FAILED output for run $run_id:" >> "$LOG_FILE"
        cat "$temp_log" >> "$LOG_FILE"
        echo "---" >> "$LOG_FILE"
    fi

    rm -f "$temp_log"
}

# Run tests in parallel batches
completed=0
running=0

# Create a named pipe for collecting results
results_pipe=$(mktemp -u)
mkfifo "$results_pipe"

# Start collecting results in background
exec 3<>"$results_pipe"
rm "$results_pipe"

# Collect results
while (( completed < RUNS )); do
    # Start new tests if we have capacity
    while (( running < PARALLEL && completed + running < RUNS )); do
        run_id=$((completed + running + 1))
        run_test "$run_id" &
        ((running++))
    done

    # Wait for a result
    if read -t 1 result <&3; then
        IFS=':' read -r run_id status runtime <<< "$result"

        if [[ "$status" == "PASS" ]]; then
            ((passed_runs++))
        else
            ((failed_runs++))
        fi

        runtimes+=($runtime)
        ((total_runs++))
        ((running--))
        ((completed++))

        echo "Run $run_id/$RUNS: $status (${runtime}s)"

        # Progress indicator
        if (( completed % 10 == 0 )); then
            echo "Progress: $completed/$RUNS runs completed (Pass: $passed_runs, Fail: $failed_runs)"
        fi
    fi
done

# Wait for any remaining background processes
wait

end_time=$(date +%s)
total_duration=$((end_time - start_time))

echo
echo "=== STRESS TEST RESULTS ==="
echo "Total runs: $total_runs"
echo "Passed: $passed_runs"
echo "Failed: $failed_runs"
echo "Success rate: $((passed_runs * 100 / total_runs))%"
echo "Total duration: ${total_duration}s"
echo "Average runtime per run: $(echo "scale=3; $total_duration / $total_runs" | bc -l)s"
echo

# Calculate median runtime
IFS=$'\n' sorted=($(sort -n <<<"${runtimes[*]}"))
unset IFS
median_index=$((total_runs / 2))
if (( total_runs % 2 == 0 )); then
    median1=${sorted[$((median_index-1))]}
    median2=${sorted[$median_index]}
    median=$(echo "scale=3; ($median1 + $median2) / 2" | bc -l)
else
    median=${sorted[$median_index]}
fi

# Handle edge case for array access
if (( total_runs > 0 )); then
    max_index=$((total_runs - 1))
else
    max_index=0
fi

echo "Runtime statistics:"
echo "  Min: ${sorted[0]}s"
echo "  Median: ${median}s"
echo "  Max: ${sorted[$max_index]}s"
echo

# Calculate percentiles
p95_index=$((total_runs * 95 / 100))
p99_index=$((total_runs * 99 / 100))
if (( p95_index >= total_runs )); then p95_index=$max_index; fi
if (( p99_index >= total_runs )); then p99_index=$max_index; fi
echo "Percentiles:"
echo "  95th: ${sorted[$p95_index]}s"
echo "  99th: ${sorted[$p99_index]}s"
echo

if (( failed_runs == 0 )); then
    echo "🎉 ALL TESTS PASSED! The E2E framework is highly stable."
else
    echo "⚠️  Some tests failed. Check $LOG_FILE for details."
    echo "Failed runs: $failed_runs"
fi

echo
echo "Detailed results saved to: $LOG_FILE"