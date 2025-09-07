# Performance Optimizations Applied

## Problem
The UI was freezing after loading ~40 repositories when using the app with large directories (82 repos).

## Root Causes Identified
1. **Batch events weren't being processed** - The eventAdapter didn't know about ReposDiscoveredBatchEvent type
2. **Excessive UpdateOrderedLists calls** - This expensive operation was being called too frequently
3. **Git operations creating event storms** - Each status update could trigger UI re-sorting

## Fixes Applied

### 1. Fixed Batch Event Processing
Added ReposDiscoveredBatchEvent to the eventAdapter.Type() method in model.go:
```go
case eventbus.ReposDiscoveredBatchEvent:
    return "eventbus.ReposDiscoveredBatchEvent"
```

### 2. Added Debouncing for UpdateOrderedLists
Created a debounced update mechanism to prevent excessive sorting:
```go
func (m *Model) debouncedUpdateOrderedLists() {
    // Cancel any existing timer
    if m.updateTimer != nil {
        m.updateTimer.Stop()
    }
    
    // Schedule a new update in 100ms
    m.updateTimer = time.AfterFunc(100*time.Millisecond, func() {
        log.Printf("Debounced UpdateOrderedLists triggered")
        m.coordinator.UpdateOrderedLists()
    })
}
```

### 3. Optimized Update Patterns
- Used debounced updates for batch repository discovery
- Kept immediate updates for user-initiated actions (sort mode changes)
- Removed UpdateOrderedLists from individual status updates

### 4. Added Performance Logging
Added timing logs to UpdateOrderedLists to monitor performance:
```go
log.Printf("UpdateOrderedLists completed in %v (repos=%d, groups=%d)", time.Since(start), len(repos), len(groups))
```

## Expected Results
- UI should remain responsive during repository loading
- All 82 repositories should load without freezing
- Navigation should be smooth after loading completes

## Testing
Run the app and monitor the logs:
```bash
./gitagrip 2>gitagrip.log
# In another terminal:
tail -f gitagrip.log | grep -E "(UpdateOrderedLists|Debounced|batch)"
```

Look for:
- Batch events being properly processed
- Debounced updates reducing frequency
- UpdateOrderedLists execution times