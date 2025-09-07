# Debug Logging Guide

I've added extensive logging throughout the application to identify the freezing issue. Here's what to look for in the logs:

## Log Prefixes

- `[FORWARD]` - Event forwarding from main.go to UI
- `[WAIT_EVENT]` - UI waiting for and receiving events
- `[UPDATE]` - UI Update() method processing
- `[HANDLE_DOMAIN]` - Domain event handling
- `[VIEW]` - UI View() rendering
- `[GIT_SERVICE]` - Git operations and status updates
- `[CHANNEL_MONITOR]` - Event channel status every 5 seconds

## Key Things to Look For

1. **Event Channel Backlog**
   - Look for `[CHANNEL_MONITOR]` messages showing channel filling up
   - If channel reaches 1000/1000, events are being dropped

2. **Update/View Pattern**
   - Each event should trigger: `[UPDATE]` → `[HANDLE_DOMAIN]` → `[VIEW]`
   - Look for missing steps or very slow operations

3. **Git Service Progress**
   - `[GIT_SERVICE] Processing repo X/82` shows progress
   - Each repo generates a status update event

4. **Performance Warnings**
   - `[VIEW] WARNING: View() took Xms` if rendering is slow
   - `[HANDLE_DOMAIN] Completed in X` shows event processing time

5. **Event Flow**
   - `[FORWARD]` should match `[WAIT_EVENT]` received events
   - No events should be re-published (no loops)

## What Might Be Wrong

1. **Event Storm**: 82 repos × 4 git commands = 328 status updates flooding the UI
2. **Blocking Operations**: Look for gaps in timestamps indicating blocking
3. **Channel Overflow**: Events being dropped due to full channel
4. **Render Performance**: View() taking too long with many repos

Run the app with: `./gitagrip 2>gitagrip.log`
Then check the log to see where it gets stuck.