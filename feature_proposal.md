YARG – Multi-Repo Git TUI (Initial Feature Set Proposal)
Core MVP Features (Read-Only Operations)
Recursive Repository Discovery: YARG will recursively scan a user-specified base directory to auto-detect all Git repositories (by locating .git/ folders). This provides an immediate overview of multiple repos without manual listing. The discovery should skip descending into a repo’s own subdirectories once a .git is found (to avoid treating sub-repos or submodules twice). Scanning will be done efficiently – e.g. using the walkdir crate or similar – and can be parallelized for speed (using threads or Rayon) so the TUI isn’t blocked during startup
github.com
. The base directory can be provided via a CLI argument or a default in config.
Repository Status Overview: The main TUI view will list each discovered repository along with key read-only status info:
Name/Path: Show the repo name (folder name) or path for identification.
Current Branch: Display the current checked-out branch name for each repo.
Dirty/Clean Indicator: Indicate if the working directory has uncommitted changes or untracked files (e.g. an icon or color to flag a dirty repo vs. clean git status).
Ahead/Behind: Optionally show if the local branch is ahead/behind its remote (requires having fetched, see below).
This gives a compact “dashboard” similar to a condensed git status for all repos. All data is read-only (no modifications to repos). Information can be gathered via a Git library (like git2-rs) without shelling out to git for performance and reliability
github.com
.
Read-Only Git Actions: Provide a few on-demand actions that retrieve info without altering repository state:
Fetch: Allow the user to trigger a git fetch for one or all repositories to update their remote tracking info (no merge or pull, just fetch). This lets YARG update ahead/behind status indicators. Fetch operations will be run in the background (per repo or batch) so the UI remains responsive.
View Commit Log: When a specific repo is selected, the user can open a read-only view of recent commits (e.g. last N commit messages on the current branch). This could be a popup or separate panel showing commit hash, author, date, and message for each of the latest commits (similar to a git log --oneline). This helps inspect history without leaving the TUI. Scrolling/paging through log entries should be supported if the list is long.
Branch List (optional): As a non-editable reference, the tool might allow viewing the list of local branches and perhaps the active one marked. (This could be in a future version if not in MVP, unless easily fetched via git2.)
Grouping of Repositories: To manage many repos ergonomically, YARG will support grouping repositories:
Automatic Grouping: By default, YARG can group repos based on directory structure. For example, all repos under the same immediate parent folder could be shown under a collapsible group heading named after that folder. This gives an automatic organizational hierarchy without user configuration (useful if your projects are organized by client or category folders).
Manual Grouping: Users can define custom groups in a config TOML to override or supplement the auto-grouping. For instance, you might group disparate directories into a “Work” group vs “Personal” group. The config file lets you assign specific repo paths to named groups. In the UI, those groups will be displayed (and can be collapsed/expanded). Any repos not covered by a manual group can fall back to an “Ungrouped” or auto-group category. This dual approach (auto + manual) ensures flexibility in organizing repos.
UI Behavior: Groups in the TUI will act as sections – e.g. selectable headers that can collapse to hide the repos within. This keeps the interface tidy for dozens of repos. Arrow keys or hotkeys (like Enter or Space) can toggle group expand/collapse.
TOML Configuration Persistence: YARG will persist configuration in a TOML file (e.g. yarg.toml in the user’s config directory following XDG standards). The config serves two purposes:
Repo & Group Persistence: After the initial scan, the list of repos and any manual grouping definitions can be saved. This means on the next launch YARG can load known repositories from the config instantly (skipping a full rescan unless requested) and remember custom group assignments. The config might list groups and their member repo paths, or store each repo with an associated group. Using TOML with serde makes it easy to serialize/deserialize this info in an idiomatic Rust way
lib.rs
.
User Preferences: The config can also hold user settings (like a default base directory to scan, interface preferences, etc.) so the tool is customizable. For MVP, the main config focus is storing the repository list and grouping, but the file is extensible for future settings.
Note: If no config exists on launch, YARG can perform a fresh scan and then optionally offer to save that to a new config (or do so automatically). This way first-time use requires minimal setup, but subsequent uses benefit from persistence.
User Interface & Navigation: The TUI (built with Ratatui) should follow ergonomic design principles for clarity and responsiveness:
Use a clear layout (for example, a single main panel listing groups and repos, plus a status bar for messages; or a two-panel view where one is a list and another shows details for the selected repo).
Navigation: Support keyboard navigation (e.g. Up/Down arrows or j/k to move through repo list; Right/Left or Enter to expand/collapse groups; q to quit). The interface should highlight the currently selected repository or group for context.
Action Keys: Provide intuitive keys for actions like refresh (r key to rescan/refresh statuses), fetch (f to fetch updates for the selected repo or all repos), log view (l to open the commit log popup for the selection), etc. These keys and their functions should be discoverable – for instance, a small help menu or hint bar at the bottom can list the key bindings.
Non-blocking UI: Ensure that lengthy operations (scanning, fetching, log loading) do not freeze the TUI. This can be achieved by spawning background threads or tasks for those operations and then updating the UI state via message passing when they complete. This design aligns with Ratatui’s immediate mode architecture, where the main thread should primarily handle drawing and input events
github.com
.
Visual Feedback: Give the user feedback for actions – e.g. a status line message like “Fetching origin…done” or a loading spinner while scanning is in progress. Color-coding can be used to convey repo state (for example, green for clean repo, yellow for untracked changes, red for uncommitted changes or errors, etc., and perhaps different colors if a repo is ahead/behind remote). This makes the dashboard glanceable and user-friendly.
Minimal Dependence on Mouse: The TUI will be primarily keyboard-driven (which is idiomatic for terminal apps), though basic mouse support (click to select or expand a group) could be considered later if Ratatui supports it. MVP will focus on keyboard control.
Architecture and Module Design
Modular Code Structure: Structure the project into clear modules separating concerns:
Config Module: Handles reading/writing the TOML config. Define data structs (using serde::Deserialize/Serialize) for things like repository entries and group definitions. This module abstracts persistence – e.g., Config::load(file_path) and Config::save(...) functions. Using Serde + TOML is idiomatic for Rust apps to manage config files
lib.rs
.
Discovery Module: Responsible for scanning the filesystem for git repositories. This can provide a function like find_repos(base_path) -> Vec<Repo> that returns a list of discovered repos (each maybe as a struct with basic info). It can use crates like walkdir or ignore (to respect .gitignore files if needed) to efficiently traverse directories. Make this module ignorant of UI – it just yields data. If the scan is potentially slow, consider running it in a thread and perhaps emitting progress updates.
Git Operations Module: Wrap Git-related queries using a library (e.g. git2 crate for direct repository access). Functions here might include get_repo_status(repo_path) returning branch name, dirty status, etc., fetch_repo(repo_path), and get_recent_commits(repo_path, n) for the log. Using the git2 library is idiomatic and avoids having to spawn subprocesses for every operation
lib.rs
. It also keeps the app truly read-only (no accidental working dir changes). That said, for operations like fetch, an external git command could be invoked if libgit2 is complex for networking; this module can abstract that choice. All functions should return Result<…> with proper error handling (perhaps using anyhow or a custom error type via thiserror for clarity).
UI Module: Contains the Ratatui-based rendering code and event handling. Likely further split into subcomponents:
e.g. ui::draw_main(state: &AppState, frame: &mut Frame): draws the main repo list grouped as needed, and maybe other sections.
e.g. ui::draw_log_popup(state: &AppState, frame: &mut Frame): draws the commit log overlay when activated.
This module will use Ratatui widgets like List or Table for the repo list, Block for group sections, etc., styling with borders, colors, etc., per ergonomic TUI guidelines. By isolating drawing code, it’s easier to tweak the interface without affecting logic.
App State & Event Loop: Design an AppState struct that holds all dynamic data: list of repositories (with their statuses), grouping info, currently selected item indices (which group and which repo within that group), and flags for UI modes (e.g. whether the log popup is open, whether a fetch is in progress, etc.). The event loop (in main.rs or a controller module) will:
Initialize by loading config and/or scanning repos.
Spawn any worker threads needed (for scan or periodic refresh).
Enter a loop to process user input events (using e.g. Crossterm for key events) and update AppState accordingly, then call the UI draw routine each tick to reflect changes.
Use channels (e.g. std::sync::mpsc or crossbeam_channel) for background threads to send results back (like “repo X status fetched”). The main loop can select on these to merge updates into AppState.
This architecture follows typical idiomatic Rust TUI patterns: the UI rendering is driven by a single state object, and all mutating actions funnel through the event loop (either from user input or background messages). It keeps the code simpler and avoids concurrency issues by mostly updating state on one thread.
Rust Best Practices & Ergonomics:
Leverage strong typing: define a Repository struct (with fields like path, name, current_branch, status_summary, group_name, etc.) to pass around instead of raw strings. This makes functions self-documenting (e.g., fn fetch_repo(repo: &Repository) rather than passing just a path).
Implement trait Display or similar for nice formatting of repository status if needed (could be used for logging or debug).
Use error handling conscientiously: propagate errors from git operations to the UI – for example, if a fetch fails (network down, etc.), catch that and display an error message to the user instead of panicking. Employing Result and the ? operator will keep error handling idiomatic. A central error type (using thiserror) could encapsulate different error kinds (IO, Git, Config parse) for unified handling.
Follow the Rust community’s style for config and CLI: for instance, consider using clap to parse a command-line argument for the base directory or config file path if provided (even in a TUI app, CLI options for config can be useful)
lib.rs
. This makes the tool scriptable and user-friendly.
Write unit tests for non-UI logic (e.g. the grouping logic function, config load/save roundtrip, parsing of git status outcomes if any). Keeping logic in separate modules (as above) helps test them without needing the TUI, which is good Rust development practice.
Ergonomic TUI Design Considerations:
Keep the interface clean: avoid overloading the screen with too much text. The MVP should probably show one line per repo in the list. Additional details (like the exact list of modified files or commit diffs) can be left for later versions or on-demand views. This prevents clutter and aligns with the principle of making a TUI that “does one job well” initially.
Use spacing, borders, and text alignment to improve readability. For example, Ratatui’s layout can divide the terminal into chunks – perhaps a top area for a title or path of the base directory, a large center pane for the repository list, and a single-line footer for help or status messages.
Color and text style (bold for headings, inverse video for selected item, etc.) should be used consistently to guide the user’s eyes. Each repo line might display branch names in a different color, or show a symbol like ● for dirty (in red) vs ✔ for clean (in green), etc., to convey status at a glance.
Ensure the TUI handles common terminal scenarios: window resize (the layout should recalibrate on size changes), and graceful exit (restoring the cursor and terminal state on quit, which Ratatui/Crossterm can handle if dropped correctly). These little details make the app feel polished.
Provide a quick reference (maybe press ? to open a help dialog listing keys). This is especially helpful as features grow, but even in MVP a small static help screen or section can improve UX for new users.
Feature Prioritization and Future Enhancements
MVP Focus: The initial version of YARG should prioritize robustness and usability of the core features listed above. In particular, getting the multi-repo scanning and status display right is the top goal – this is the essence of the tool (aggregating read-only info from many repos). The grouping and config persistence are also core requirements, as they improve organization and allow the app to be more than just a one-off scanner (they add state and customization to the experience). It’s important that MVP handles these basics in an idiomatic way (e.g. concurrency for performance, config via serde, non-blocking UI). Features like fetch and log viewing are desirable but should be implemented only after the basic repository list view is working smoothly. If time is tight, the commit log viewer could be a candidate to simplify (e.g., MVP might show only the latest commit message in the list, and defer a full interactive log until a later update). Post-MVP Ideas: Once the above foundation is solid, YARG can gradually add more advanced capabilities:
Interactive Git Actions: Future versions can move beyond read-only. For example, allowing git pull or git switch branch from the UI, staging and committing changes, or opening an external editor for a file. These should be added cautiously (after MVP) to ensure the tool remains stable; each write-operation would need confirmations to avoid mistakes.
Bulk Operations: Ability to run an action on multiple repos at once (e.g., fetch all, pull all, open all in editor, etc.). This can leverage the grouping – e.g. “fetch every repo in the Work group with one command”.
Filtering/Searching: As the number of repos grows, adding a filter box to quickly find a repository by name/path could be useful. This might involve a text input in the UI (which Ratatui can handle) and filtering the list dynamically.
Enhanced Group Management: In the MVP, groups are mainly managed via config. Later, a TUI interface to create/delete groups and move repos between groups interactively would be nice. This could be a “group management mode” or done through a help menu.
Status Details: Showing which files are changed (not just that the repo is dirty) possibly in a sub-view, or integrating a diff viewer for a selected repo’s changes. This starts to overlap with full TUI git clients (like a simplified Magit or lazygit inside YARG) – useful, but should come after the multi-repo overview functionality is perfected.
Notifications/Watch: The app could watch repositories for changes (using file system notifications) and update status in real-time. For example, if you edit files or if new commits arrive (after a fetch) it could auto-refresh that repo’s info. This is advanced (requires async FS watching or polling) and can be considered in future to make the dashboard “live”.
Performance Scaling: Future improvements might involve caching and optimizing for a very large number of repos. MVP will handle, say, dozens of repos easily; but if a user has hundreds, further optimizations or UI virtualization (rendering only visible items) might be needed. Using efficient data structures and avoiding re-calculating unchanged info will become important as a later enhancement.
All these future enhancements should be designed in harmony with the idiomatic Rust architecture established in the MVP. By laying a good foundation (clear module boundaries, immutable vs mutable state handling, etc.), YARG will be well-positioned to grow in functionality without becoming unmanageable. In summary, the MVP should nail the fundamentals (multi-repo discovery, status, grouping, config), and subsequent versions can iterate with more interactive and power-user features, guided by community feedback and Rust’s best practices. The result will be a ergonomic terminal UI tool that feels intuitive for users and is maintainable for developers. Sources:
Nick Gerace, gfold – A CLI tool for tracking multiple Git repos (illustrating concurrent repo scanning and use of git2 for read-only analysis)
github.com
.
Hannes Körber, git-repo-manager (GRM) – A config-driven Git manager (demonstrates use of TOML/Serde for config and git2 library for Git operations in Rust)
lib.rs
.
Citations

GitHub - nickgerace/gfold: CLI tool to help keep track of your Git repositories, written in Rust

https://github.com/nickgerace/gfold

git-repo-manager — Rust utility // Lib.rs

https://lib.rs/crates/git-repo-manager
