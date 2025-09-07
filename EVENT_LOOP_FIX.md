# Event Loop Fix Summary

## Root Cause
The app was freezing due to an infinite event loop:
1. Domain events arrived from main.go via eventChan
2. UI's handleDomainEvent was re-publishing these events to the internal UI event bus
3. The UI had subscriptions to these same event types on its internal bus
4. This created a feedback loop where events kept getting re-published

## Symptoms
- UI became sluggish then froze after ~40 repositories
- Input lag increased before freezing
- Logs showed continuous "UI: Update processing eventReceivedMsg" followed by "EventBus: Publishing event logic.RepositoryUpdatedEvent"
- Event processing consumed all CPU, starving the input handler

## Solution
1. **Removed event re-publishing**: handleDomainEvent now processes events directly instead of publishing to the internal event bus
2. **Direct event handling**: Each domain event type is handled immediately in handleDomainEvent
3. **Separated concerns**: The internal UI event bus is reserved for UI service communication only

## Additional Optimizations
1. **Debounced UpdateOrderedLists**: Added 100ms debouncing to prevent excessive sorting
2. **Reduced git worker pool**: Limited to 2 concurrent operations (from 5)
3. **Added rate limiting**: 50ms delay between repos, 200ms every 5 repos
4. **Fixed batch processing**: Batch events now processed correctly

## Code Changes
- Modified `handleDomainEvent` to process events directly
- Added `handleReposDiscoveredBatch` for batch processing
- Removed domain event subscriptions from `subscribeToEvents`
- Added timing logs to track performance

## Result
The app should now:
- Handle 82+ repositories without freezing
- Maintain responsive input handling
- Process git status updates without blocking the UI
- Show all repositories progressively as they load