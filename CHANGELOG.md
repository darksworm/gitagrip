# Changelog

## [2.0.0](https://github.com/darksworm/gitagrip/compare/v1.1.0...v2.0.0) (2025-10-03)


### ⚠ BREAKING CHANGES

* **keybinds:** Enter no longer opens commit history; use Shift+H instead.

### Features

* Add log and diff viewer functionality ([3bc5bdf](https://github.com/darksworm/gitagrip/commit/3bc5bdf7db2382520604d4df2dab5cc06fb57d09))
* add log-diff viewer with ov pager ([2ac5731](https://github.com/darksworm/gitagrip/commit/2ac5731fc52127ca486cda23ec06a533b27df3bd))
* Add production-grade Go TUI testing framework ([530b2ff](https://github.com/darksworm/gitagrip/commit/530b2ff6639de5b1aafae647fed21391d10e4e53))
* **branch:** add new-branch (b) and switch-branch (s) flows; remap sort to Shift+S ([ead40b0](https://github.com/darksworm/gitagrip/commit/ead40b0dd697d0644101e893429804b3cb9ab33f))
* Convert help system to ov pager with vim-style navigation ([a319d56](https://github.com/darksworm/gitagrip/commit/a319d56fdeca2eb69e94b34d0bd85c0d4f975639))
* don't open diff pager when there are no changes ([3d03299](https://github.com/darksworm/gitagrip/commit/3d0329945fc42b00074f1dba723b010e84048836))
* enhance key bindings and add git operations tests ([3d385e8](https://github.com/darksworm/gitagrip/commit/3d385e8d81807b49bc5bd0b498cc9dc972b2ff18))
* ensure Bubble Tea rendering is paused during ov pager ([b05f966](https://github.com/darksworm/gitagrip/commit/b05f96649843a9dd3e5b876177b6f02b175dac76))
* **keybinds:** open git log with Shift+H; Enter no longer opens history ([3fa6680](https://github.com/darksworm/gitagrip/commit/3fa66808fe40d3c9c173f5b7dc531688bb2ff2a6))
* **lazygit:** launch lazygit on Enter and pause rendering ([1f07600](https://github.com/darksworm/gitagrip/commit/1f076004e1234d6a55825a2f4c76f64063e3de44))
* make status messages more visible and auto-clearing ([e3a1b2e](https://github.com/darksworm/gitagrip/commit/e3a1b2efcc8debeb78034d8e49f09319caffbbeb))
* Optimize E2E tests with event-driven waiting and ring buffer ([b01baf9](https://github.com/darksworm/gitagrip/commit/b01baf9b771c6784d92cfedda23dd2b3894fc877))
* **ui/modals:** upgrade to Bubble Tea/Lipgloss v2 betas and use layered grayscale modals ([30bc9f6](https://github.com/darksworm/gitagrip/commit/30bc9f66248882601d31e4bc88f657d1645dc8b5))
* **ui:** add per-repo logs viewer (Shift+I) and info cleanup; avoid flooding status bar with errors ([3ef7a32](https://github.com/darksworm/gitagrip/commit/3ef7a322224bb6c6e2804e9fa76ccdeb665f1f01))


### Bug Fixes

* Handle git diff exit code 1 correctly ([12f5022](https://github.com/darksworm/gitagrip/commit/12f50221374d24b9e7d4c5b12733e15c51eb13cd))
* handle unchecked error returns in gitops.go ([a250b52](https://github.com/darksworm/gitagrip/commit/a250b52b8f128ee04ae6e1372365e681c0ab094a))
* reinstate 'd' binding for deleting groups ([cca3fba](https://github.com/darksworm/gitagrip/commit/cca3fba07311bc79ed2710854963e5a260e2e3cb))
* render status message when no diff ([f3fb59c](https://github.com/darksworm/gitagrip/commit/f3fb59c8a64f58bab2c59074217f72f8c37dd97d))
* restore vim-style navigation for 'l' key ([d461c9c](https://github.com/darksworm/gitagrip/commit/d461c9c2471a6816a9e5c8366b696dab4d7e10ed))
* **ui:** stop surfacing error text in status bar; show error icon and log details ([737c45f](https://github.com/darksworm/gitagrip/commit/737c45fbe2c5cb20301f5a1fe624bceb07383a6a))

## [1.1.0](https://github.com/darksworm/gitagrip/compare/v1.0.6...v1.1.0) (2025-09-07)


### Features

* highlight group header when all repos in group are selected ([9789611](https://github.com/darksworm/gitagrip/commit/97896119625103f8e098f8a6fa444b013ab4420d))


### Bug Fixes

* improve cursor visibility on selected items ([a947b5a](https://github.com/darksworm/gitagrip/commit/a947b5ad8776782da42020bcc6a729732726bcd6))

## [1.0.6](https://github.com/darksworm/gitagrip/compare/v1.0.5...v1.0.6) (2025-09-07)


### Bug Fixes

* correct CLI usage documentation and simplify Docker usage ([08691e1](https://github.com/darksworm/gitagrip/commit/08691e147ee43d3aa968b4f183e7860e4dbbfbc4))

## [1.0.5](https://github.com/darksworm/gitagrip/compare/v1.0.4...v1.0.5) (2025-09-07)


### Bug Fixes

* add QEMU setup for ARM64 Docker builds ([c451066](https://github.com/darksworm/gitagrip/commit/c4510668b19f057dc508f6eb6d4fea89fe735733))
* enable buildx for multi-arch Docker builds ([41ab83f](https://github.com/darksworm/gitagrip/commit/41ab83f578a2f00c2b8b830d5c2610d16951fd91))

## [1.0.4](https://github.com/darksworm/gitagrip/compare/v1.0.3...v1.0.4) (2025-09-07)


### Bug Fixes

* add platform argument to Dockerfile for multi-arch builds ([fdb7c18](https://github.com/darksworm/gitagrip/commit/fdb7c1871c13c43a0d2b6b85a4f7e2101c8dcd57))
* update govulncheck to latest version ([a4eb4ba](https://github.com/darksworm/gitagrip/commit/a4eb4bad57b76917f9b72e98016ba4cedc24d78d))
* update staticcheck to 2025.1 ([ca48545](https://github.com/darksworm/gitagrip/commit/ca48545e9e68181908472962a360b2b3d3dbcdf0))

## [1.0.3](https://github.com/darksworm/gitagrip/compare/v1.0.2...v1.0.3) (2025-09-07)


### Bug Fixes

* simplify Dockerfile for GoReleaser builds ([65ec894](https://github.com/darksworm/gitagrip/commit/65ec89453ac1b1ae58ce7f4698812eae6ed3523e))

## [1.0.2](https://github.com/darksworm/gitagrip/compare/v1.0.1...v1.0.2) (2025-09-07)


### Bug Fixes

* change homebrew folder to directory field ([f47f3b1](https://github.com/darksworm/gitagrip/commit/f47f3b1dd07c4b8041bee5a81ed5a7fba21b75a9))

## [1.0.1](https://github.com/darksworm/gitagrip/compare/v1.0.0...v1.0.1) (2025-09-07)


### Bug Fixes

* trigger release process ([c92b2e2](https://github.com/darksworm/gitagrip/commit/c92b2e2d3eef03b3b4ca2b85344f698c48f7271b))

## 1.0.0 (2025-09-07)


### Features

* Add basic group management functionality ([0b30366](https://github.com/darksworm/gitagrip/commit/0b30366072992d63c2d2a5e96ae68386654ec39b))
* Add CLI args and auto-generate groups from directory structure ([e20e77e](https://github.com/darksworm/gitagrip/commit/e20e77eabee1d6a0c2e594158aaa81db48893cad))
* Add commit log viewer with 'l' key ([6aef762](https://github.com/darksworm/gitagrip/commit/6aef762118b0a7f48e28ccbbc5f0b8397d1fc2e1))
* Add comprehensive features to GitaGrip TUI ([8d854ca](https://github.com/darksworm/gitagrip/commit/8d854ca069abe614c1577f995142c9ee3a535ecc))
* Add comprehensive keyboard navigation ([6a02904](https://github.com/darksworm/gitagrip/commit/6a02904c8ffb2d5120ac70b8328d279559122eb4))
* Add cross-group navigation with { and } keys ([6b2bc1c](https://github.com/darksworm/gitagrip/commit/6b2bc1c23959a828abbebdcda2eca65a175122fe))
* Add ESC to clear selection and auto-expand groups for search/filter ([e2b5e38](https://github.com/darksworm/gitagrip/commit/e2b5e38509fb4b04da66f44afff98634aebdf4b6))
* Add fetch operation with loading indicators ([25d5407](https://github.com/darksworm/gitagrip/commit/25d5407ebd01ca6d91bd116ad2414ba675190087))
* Add group deletion with 'd' key ([3ea46b7](https://github.com/darksworm/gitagrip/commit/3ea46b7a782bd0bce03cf18acde915e3ae376aaa))
* Add group renaming with Shift+R and remove full scan ([dd97dda](https://github.com/darksworm/gitagrip/commit/dd97ddaa2d3d3a98d3a29e34f213ccb06a1ae448))
* Add group reordering with Shift+J/K and Shift+Arrow keys ([70edcdb](https://github.com/darksworm/gitagrip/commit/70edcdbe282b35c3ac87a8af4a234f80bf10e76e))
* Add interactive sort mode and hide functionality ([c54d901](https://github.com/darksworm/gitagrip/commit/c54d90155c4b5ddb3be278c773e2c368292f202b))
* Add loading screen and improve UI highlighting ([3475c4f](https://github.com/darksworm/gitagrip/commit/3475c4f935788bf319cc2eb9a99bcfa9817b7942))
* Add multi-select and refresh functionality with loading states ([9ae52e0](https://github.com/darksworm/gitagrip/commit/9ae52e02d25d97357a6c988e9faad821dea87ca5))
* Add progress indicators for batch operations ([c103e74](https://github.com/darksworm/gitagrip/commit/c103e7434cd7266f073f94d881301f657c9828a8))
* Add scanning progress indicators in UI ([0e2f436](https://github.com/darksworm/gitagrip/commit/0e2f4363b4379591feaa8c77bbdf62658b589780))
* add scrollable repository list with keyboard navigation ([76467c9](https://github.com/darksworm/gitagrip/commit/76467c9cc274ac46d3d167fcce7cd6ede6213a5d))
* Add viewport scrolling and full-row highlighting ([6b97a29](https://github.com/darksworm/gitagrip/commit/6b97a29544256529a1cc25971d99cc2f82be3a0c))
* Add vim-style search with / key ([6cccfff](https://github.com/darksworm/gitagrip/commit/6cccfff1d7d7da0c314a22d15794bfbe3c971cff))
* Add visual gaps between groups and handle duplicate repository names ([bc13528](https://github.com/darksworm/gitagrip/commit/bc13528d0dd4ff39a6e9aaf55225263d24f545e9))
* Change logo to lowercase and add group order persistence ([91f7f1f](https://github.com/darksworm/gitagrip/commit/91f7f1fec0ca8e31305dbda36d1db1cf8bf87df6))
* colored branch name ([f5a6d63](https://github.com/darksworm/gitagrip/commit/f5a6d63d16f65b99d847149c33eca7ca239fdae2))
* complete M0 milestone with basic TUI and guiding star test ([42a94fd](https://github.com/darksworm/gitagrip/commit/42a94fd0cde0cbe5c17a55db8672dc25b49dc9ec))
* complete M1 milestone with config and CLI support ([730969c](https://github.com/darksworm/gitagrip/commit/730969cc361bdbc6877d840a4b6d50324f4df33b))
* complete M2 milestone with repository discovery and background scanning ([4c10eaf](https://github.com/darksworm/gitagrip/commit/4c10eaf26e6130cf75dd5450f61f5d64b2f2421d))
* complete M3 milestone with git status integration ([4eb9cdd](https://github.com/darksworm/gitagrip/commit/4eb9cdd3ed4149d2fb29f49cb2c1a0d93000a5af))
* Convert config format from JSON to TOML ([ab10f40](https://github.com/darksworm/gitagrip/commit/ab10f401be6c4099ebd270385620cfcd6091711b))
* Display groups at the top of the repository list ([cf2559f](https://github.com/darksworm/gitagrip/commit/cf2559f09be674948cee7b58a2477a2483558f5e))
* Enhance organize mode UX and fix move functionality ([b59c28c](https://github.com/darksworm/gitagrip/commit/b59c28c5d9c61d63a461055d91615d21ad0ec07a))
* Enhance UI with error tracking, improved visuals, and better layout ([10c0bfd](https://github.com/darksworm/gitagrip/commit/10c0bfdf9d1a37e1b5f253c866662b131c45c3f3))
* Expand branch color palette for better variety ([c141215](https://github.com/darksworm/gitagrip/commit/c141215875ec8d70233b2e7ce745bd9768a75eb7))
* Fix organize mode selection mismatch and add vim navigation ([07f3524](https://github.com/darksworm/gitagrip/commit/07f3524db3cba4d668fd6f38c0985e954b9d778d))
* Implement full row highlighting for repository selection ([c5c24d8](https://github.com/darksworm/gitagrip/commit/c5c24d898294154f9788756c9a21c4226b91fd85))
* Implement M4 modal architecture foundation ([a9a641d](https://github.com/darksworm/gitagrip/commit/a9a641d768c53a2c0536267236576e83af7c8897))
* Implement Phase 2 - Repository Selection and Movement ([221ec87](https://github.com/darksworm/gitagrip/commit/221ec876a8613c3b1f93e6eae6c04ce111e3aa70))
* Move all input prompts to top of screen ([feb1171](https://github.com/darksworm/gitagrip/commit/feb117125d26a8c6c6c923477923dfd553ecbe17))
* rename YARG to GitaGrip ([fac7ddf](https://github.com/darksworm/gitagrip/commit/fac7ddfc6263e282b0f051bb8c1fb6cef5e7529f))
* Require selection for new group creation ([4dbaa6b](https://github.com/darksworm/gitagrip/commit/4dbaa6baac0edbd4ca098ef8bc85d3d4657eac1c))
* Rewrite GitaGrip in Go with event-driven architecture ([385e44b](https://github.com/darksworm/gitagrip/commit/385e44b467cfef2bb8d5d523d00f3db894fa0903))
* Swap f and F key bindings ([185c992](https://github.com/darksworm/gitagrip/commit/185c9920526b896db729a4f8bf0ea16b580cea02))
* Use space key to toggle group expansion ([195da11](https://github.com/darksworm/gitagrip/commit/195da11f3fc356b6d8b406371348cbcf044e81bb))


### Bug Fixes

* Add missing main.go and fix .gitignore pattern ([40c74d3](https://github.com/darksworm/gitagrip/commit/40c74d3ab75df8533c9d790671c9cce0fb49820a))
* Complete input handling fixes for new input module ([9398c11](https://github.com/darksworm/gitagrip/commit/9398c11ab6931c66c79312cbe7f0ffa050178d87))
* eliminate UI jittering with stable repository ordering ([de40305](https://github.com/darksworm/gitagrip/commit/de4030510dafc8bbf088adbc8ff1d9e3e01a38fb))
* Enable fetch and pull commands on group headers ([255aabf](https://github.com/darksworm/gitagrip/commit/255aabfdc5350c0188a659e939cbf9f794ed9598))
* Enable group operations from any repo within the group ([99fde7a](https://github.com/darksworm/gitagrip/commit/99fde7a2f01e183810c1bf650934c1b4c87d47f9))
* Fix duplicate prompts and empty delete confirm message ([1577c5d](https://github.com/darksworm/gitagrip/commit/1577c5d3e4cb797f289e5b8fdfda4de978e9d6f2))
* Fix empty text input value in new group mode ([6904d70](https://github.com/darksworm/gitagrip/commit/6904d700bf3ed463ab8764d8c0804a99fdcff8dd))
* Fix group creation UI rendering issue ([ad94a97](https://github.com/darksworm/gitagrip/commit/ad94a97891479621fee1e33038f71cce59063f93))
* Fix input handling issues in new input module ([6907107](https://github.com/darksworm/gitagrip/commit/69071078ff41c2c7bbcdfdf7261485d7c7fb68ac))
* Fix scroll viewport calculations and off-by-one errors ([7018d95](https://github.com/darksworm/gitagrip/commit/7018d95f34a80a1ef8af3fb2b773e49af5f9efc9))
* Fix search navigation to find ungrouped repositories ([359724c](https://github.com/darksworm/gitagrip/commit/359724c17e7a3380befda2b65cd445439205b231))
* Fix search navigation to properly track visible matches ([f3c803f](https://github.com/darksworm/gitagrip/commit/f3c803f3e83bb932eb6f668afa43e37f7c0f1bca))
* Fix text input not updating in new group creation ([a31d88e](https://github.com/darksworm/gitagrip/commit/a31d88e01fb2762622afd8ebfe06f63337ad6b67))
* Fix text input value being empty by using pointer to textinput.Model ([8396994](https://github.com/darksworm/gitagrip/commit/8396994f1989555561d68b8b6bb5fa02517aceff))
* Group expand/collapse not working on all groups consistently ([6b9c410](https://github.com/darksworm/gitagrip/commit/6b9c410ee520c280aa1eb888254ac454a8b7233b))
* Implement search functionality for n/N navigation ([0d4b767](https://github.com/darksworm/gitagrip/commit/0d4b767344d0488a00279fe0ae70366c2332a65f))
* Improve gap handling and allow group toggle from repos ([6d41e84](https://github.com/darksworm/gitagrip/commit/6d41e84a75d65b85046a6f2484d4dd21c156104a))
* Improve group creation workflow ([95a3538](https://github.com/darksworm/gitagrip/commit/95a35383d3e1db2486a8e3e6de4b498c7a0d4429))
* Improve UI with help popup, loading indicators, and better styling ([a434195](https://github.com/darksworm/gitagrip/commit/a434195c27abe39d165ebe99bb723b4b4daba9e5))
* Loading screen not exiting properly ([d08ae34](https://github.com/darksworm/gitagrip/commit/d08ae340ddd36e28236e5b6ee6ca3f7df13772c6))
* Loading screen stuck on 'Setting up repository groups' ([f2c27e9](https://github.com/darksworm/gitagrip/commit/f2c27e93b162111b9c79dd556c95ef19385c42c0))
* Move cursor to group header when closing group from within ([2e43fba](https://github.com/darksworm/gitagrip/commit/2e43fbae8dd27446159c7c29f00a5149b79fda79))
* Multiple UI and functionality improvements ([3c7930e](https://github.com/darksworm/gitagrip/commit/3c7930e2a714909f57d433ed97d20be69c63db86))
* Optimize directory scanning for large directories ([381bd4b](https://github.com/darksworm/gitagrip/commit/381bd4bb5b3a9d12ebb2f19a3810daf90c12b6c5))
* Prevent UI jittering during repository discovery ([dc4921a](https://github.com/darksworm/gitagrip/commit/dc4921a3b1e453928402f61c06e32a78ec7f0b95))
* Properly track group creation order ([203a1d3](https://github.com/darksworm/gitagrip/commit/203a1d305ea0345d387e787f2dfa238f61e00251))
* Remove empty groups from display after disbanding ([12082bb](https://github.com/darksworm/gitagrip/commit/12082bb0fc4c9b5026638c780ba926b5023ea1c3))
* Remove loading screen and 'All operations completed' messages ([c03590c](https://github.com/darksworm/gitagrip/commit/c03590ce480a36219bef4f6a2d410222f0ae651a))
* resolve all linter and staticcheck issues ([245cc56](https://github.com/darksworm/gitagrip/commit/245cc569f1d51e53a1e50437e2b1b7e8034a854e))
* Restore n/N search navigation while keeping new group functionality ([80cf8df](https://github.com/darksworm/gitagrip/commit/80cf8dfd2f1bdbb9bdc0be532a831dba0b82330a))
* Save config when groups are created/modified and on exit ([8bd0e71](https://github.com/darksworm/gitagrip/commit/8bd0e710f19e70828c6f94f1a236acef7e62abc6))
