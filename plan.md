Build YARG: a fast Rust + Ratatui TUI to discover, group, and inspect many Git repos (read-only).

Non-Goals (v1)

No write ops (no commit/push/merge).

No network credentials mgmt.

No submodule special handling beyond status.

Tech Stack & Crates

TUI: ratatui, crossterm

Git: git2 (libgit2)

FS scan: walkdir (and ignore optional)

Concurrency: crossbeam-channel (or std mpsc) + rayon (optional)

Config: serde, toml, directories (XDG paths), serde_with

Errors/logging: anyhow, thiserror, tracing, tracing-subscriber

TOML config (XDG: ~/.config/yarg/yarg.toml):

version = 1
base_dir = "/home/user/code"   # default scan root
[ui]
show_ahead_behind = true
autosave_on_exit = true

# Manual groups
[groups.Work]
repos = [
  "/home/user/code/acme-api",
  "/home/user/code/acme-web",
]

[groups.Personal]
repos = [
  "/home/user/code/dotfiles",
]

TUI Spec (MVP)

Main View

Left: collapsible groups (Manual groups first; then “Auto: <parent>”, then “Ungrouped”).

Lines per repo: ●/✔ dirty/clean, branch, ↑n/↓m (ahead/behind), repo name (path on hover/help).

Footer hint bar: ↑↓ select ←→ expand f fetch r refresh l log ? help q quit.

Log Popup

Recent N commits (hash, date, author, subject). Scrollable. Esc to close.

Help

Keybindings cheat-sheet.

Keys

Up/Down or j/k: move

Left/Right or h/l: collapse/expand groups

r: rescan statuses (fast, no full FS walk)

F: full rescan (re-run discovery)

f: fetch selected repo (or Shift+F fetch all in group/all)

l: open log popup for selected

?: help

q: quit (and autosave if enabled)

Concurrency & Messaging

Background jobs (scan, fetch, compute status, load log) run on worker threads.

Use a command bus (channel) to submit jobs; workers send events back:

Event::RepoDiscovered(Repository)

Event::RepoStatusUpdated{ path, status }

Event::RepoFetched{ path, ok, error }

Event::RepoLogLoaded{ path, commits }

Event::ScanCompleted

The main event loop integrates events into AppState and triggers redraws.

Rule: UI thread never blocks on IO.

Implementation Milestones
M0 — Bootstrap

Deliverables

Cargo project, deps, logging init, clean exit (restore terminal).
Acceptance

yarg launches to an empty TUI with a footer and can quit cleanly.

Prompt to agent

Create a new Rust binary yarg. Add the listed dependencies. Initialize tracing with env filter. Set up a Ratatui + Crossterm app that draws a placeholder layout (title + footer). Implement clean shutdown on q and Ctrl+C.

M1 — Config (TOML) + Base Args

Deliverables

Read ~/.config/yarg/yarg.toml if exists; else create minimal default.

Optional CLI flag --base-dir <path>, overriding config.
Acceptance

On start, logs show loaded config; base dir resolved; can save back (autosave_on_exit).

Prompt

Implement config module with serde TOML structs matching the schema above. Use directories to resolve config path. Add --base-dir CLI with clap or raw args. Add Config::load()/save(). Hook into app init.

M2 — Discovery (recursive scan)

Deliverables

scan::find_repos(base_dir) -> Vec<Repository>.

Skip nested descent after finding .git/.

Populate auto_group from parent dir name.

Run discovery in background. Stream discoveries to UI.
Acceptance

When pointed at a directory with repos, they appear in UI under correct groups.

Prompt

Implement scan::find_repos using walkdir (follow dirs, depth limit optional). Detect .git dirs. For each repo, emit Event::RepoDiscovered. Collapse duplicates. Don’t block UI; run in a worker thread.

M3 — Git status aggregation (read-only)

Deliverables

git::status::read_status(path) -> RepoStatus using git2:

current branch

dirty (index/workdir)

ahead/behind vs upstream (if upstream exists)

last commit summary

Batch compute on discovered repos (parallel).
Acceptance

Repo rows show branch, dirty flag, ahead/behind, last commit (optional tooltip/secondary text).

Prompt

Implement git::status::read_status via git2: resolve HEAD ref, find upstream, compute ahead/behind with graph_ahead_behind, use statuses for dirty/untracked. Add a worker that consumes repo paths and publishes RepoStatusUpdated events.

M4 — Grouping & List UI

Deliverables

Build group list from: manual groups (config) + auto groups (parent folder) + ungrouped.

Collapsible sections; selection tracking (group, index).
Acceptance

Arrow keys move selection; groups expand/collapse; selection never panics.

Prompt

Implement grouping in app.rs: a function compute_groups(&[Repository], &Config) returning a vector of renderable groups. Implement UI list drawing with Ratatui List/Block. Add expand/collapse state per group.

M5 — Fetch & Refresh (read-only)

Deliverables

git::fetch::fetch(path) (safe: no merge, --prune optional).

Key f: fetch selected; Shift+F: fetch all in current group (and maybe all repos).

After fetch completes, recompute status for that repo/group.
Acceptance

Visual feedback while fetching; ahead/behind updates afterwards; errors surface to footer.

Prompt

Implement git::fetch::fetch via git2 remote. On keypress f, submit job(s). While jobs run, show a spinner or “Fetching…”. On completion, emit status updates or error to flash message.

M6 — Commit log popup

Deliverables

git::log::recent(path, n) to read last N commits.

Popup panel with scroll; open via l, close via Esc.
Acceptance

Log lists hash, author, date, subject. Performance is smooth on large repos.

Prompt

Implement commit log retrieval with git2::Revwalk. Add ui::log popup. Handle scroll keys (PgUp/PgDn, k/j). Integrate with event bus.

M7 — Save state & polish

Deliverables

Persist manual groups & repo membership to TOML when changed.

Autosave on exit (if enabled).

Help screen (?) with keybindings.

Footer flash messages & error handling.
Acceptance

Relaunching restores groups; help works; no UI tear-down artifacts.

Prompt

Implement config write-back for groups. Add ui::help. Add a flash(message) helper to show transient messages. Ensure terminal restoration on panic/quit.

Testing & Quality

Unit tests: grouping logic, config round-trip, status parsing edge cases (detached HEAD, no upstream).

Integration tests: create temp repos with git2 (init, commit, branch) and assert status fields.

Perf sanity: scanning large trees doesn’t freeze UI (events keep flowing).

Tracing: add spans around scan/status/fetch to diagnose stalls.

Definition of Done (MVP)

Launch, scan base dir, show grouped repo list with branch/dirty/ahead-behind.

Manual groups via TOML respected.

Fetch works (single + group/all) and updates status.

Log popup shows recent commits.

Non-blocking UI, clean exit, config persists.

Guardrails & Principles (for the agent)

Clean Code: readability over brevity; small, focused modules.

TDD-ish: write tests for pure logic (grouping/config) first - verify they fail before implementing the code. When possible and easy write e2e/integration level tests.

Use semantic commits.

YAGNI: no write ops, no PR integration, no submodule special cases (yet).

DRY: shared drawing helpers & event wiring.

Least Privilege: read-only git ops; fetch only when asked.

Refactor: after each milestone, simplify interfaces (e.g., unify job/event types).

Quick “Single Prompt” Starter (you can paste this to kick off M0–M2)

You are a senior Rust dev. Bootstrap a new Ratatui app named yarg. Add the listed dependencies in Cargo.toml. Implement config loading (~/.config/yarg/yarg.toml) with fields version, base_dir, ui, and [groups.<name>].repos. On start, read config or create defaults, then kick off a background recursive scan of base_dir using walkdir, detecting .git directories. Stream discovered repos to the UI via a crossbeam channel. The main TUI shows a placeholder list that fills as repos arrive. Ensure clean terminal teardown on quit. No blocking IO on the UI thread. Include tracing logs.

When that’s green, proceed milestone by milestone with the prompts above.
