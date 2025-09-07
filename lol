fatal error: concurrent map read and map write

goroutine 1 [running]:
internal/runtime/maps.fatal({0x104b53dd0?, 0x0?})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/runtime/panic.go:1058 +0x20
gitagrip/internal/ui/views.(*Renderer).renderRepositoryList(_, {0x9e, 0x3f, 0x14000124ed0, 0x140000c48d0, {0x1400006f140, 0x4, 0x4}, 0x0, 0x140000c48a0, ...})
	/Users/ilmars/Dev/private/gitagrip/internal/ui/views/view.go:184 +0x258
gitagrip/internal/ui/views.(*Renderer).Render(_, {0x9e, 0x3f, 0x14000124ed0, 0x140000c48d0, {0x1400006f140, 0x4, 0x4}, 0x0, 0x140000c48a0, ...})
	/Users/ilmars/Dev/private/gitagrip/internal/ui/views/view.go:89 +0xa14
gitagrip/internal/ui.(*Model).View(0x140001d6008)
	/Users/ilmars/Dev/private/gitagrip/internal/ui/model.go:200 +0xa4
github.com/charmbracelet/bubbletea.(*Program).eventLoop(0x140001563c0, {0x104bfac28?, 0x140001d6008?}, 0x14000110380)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:540 +0x6d0
github.com/charmbracelet/bubbletea.(*Program).Run(0x140001563c0)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:686 +0x958
main.main()
	/Users/ilmars/Dev/private/gitagrip/main.go:246 +0xd20

goroutine 2 [syscall]:
os/signal.signal_recv()
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/runtime/sigqueue.go:149 +0x2c
os/signal.loop()
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/signal/signal_unix.go:23 +0x1c
created by os/signal.Notify.func1.1 in goroutine 1
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/signal/signal.go:152 +0x28

goroutine 35 [chan receive]:
main.main.func1()
	/Users/ilmars/Dev/private/gitagrip/main.go:75 +0x2c
created by main.main in goroutine 1
	/Users/ilmars/Dev/private/gitagrip/main.go:74 +0x3f0

goroutine 36 [select]:
gitagrip/internal/eventbus.(*bus).dispatch(0x1400012a640)
	/Users/ilmars/Dev/private/gitagrip/internal/eventbus/eventbus.go:127 +0x98
created by gitagrip/internal/eventbus.New in goroutine 1
	/Users/ilmars/Dev/private/gitagrip/internal/eventbus/eventbus.go:79 +0xe8

goroutine 41 [select]:
github.com/charmbracelet/bubbletea.(*Program).Send(...)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:745
main.main.func12()
	/Users/ilmars/Dev/private/gitagrip/main.go:229 +0xc0
created by main.main in goroutine 1
	/Users/ilmars/Dev/private/gitagrip/main.go:226 +0xc44

goroutine 42 [select]:
github.com/charmbracelet/bubbletea.(*Program).handleSignals.func1()
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:293 +0xfc
created by github.com/charmbracelet/bubbletea.(*Program).handleSignals in goroutine 1
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:284 +0x80

goroutine 130 [runnable]:
gitagrip/internal/logic.(*MemoryRepositoryStore).GetRepository(0x14000224318?, {0x14000469950?, 0x8?})
	/Users/ilmars/Dev/private/gitagrip/internal/logic/stores.go:24 +0xc8
gitagrip/internal/ui/coordinator.(*Coordinator).wireServices.func5({0x14000469950?, 0x8?})
	/Users/ilmars/Dev/private/gitagrip/internal/ui/coordinator/coordinator.go:95 +0x3c
gitagrip/internal/ui/services/sorting.(*Service).SortRepositories.func1(0x3d?, 0x0)
	/Users/ilmars/Dev/private/gitagrip/internal/ui/services/sorting/service.go:85 +0x90
sort.partition_func({0x140000c0db0?, 0x140002a0120?}, 0x0, 0x52, 0x140000c0cc8?)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/sort/zsortfunc.go:154 +0x194
sort.pdqsort_func({0x140000c0db0?, 0x140002a0120?}, 0x18?, 0x104bbd020?, 0x140000c0d78?)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/sort/zsortfunc.go:114 +0x1b0
sort.Slice({0x104bb9500?, 0x1400029e030?}, 0x140000c0db0)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/sort/slice.go:29 +0xb8
gitagrip/internal/ui/services/sorting.(*Service).SortRepositories(0x1400012e620, {0x140002b4008, 0x52, 0x52})
	/Users/ilmars/Dev/private/gitagrip/internal/ui/services/sorting/service.go:83 +0x108
gitagrip/internal/ui/coordinator.(*Coordinator).UpdateOrderedLists(0x1400010e420)
	/Users/ilmars/Dev/private/gitagrip/internal/ui/coordinator/coordinator.go:150 +0x178
gitagrip/internal/ui.(*Model).subscribeToEvents.func1({0x104bf9f80?, 0x14000291d60?})
	/Users/ilmars/Dev/private/gitagrip/internal/ui/model.go:473 +0xc8
gitagrip/internal/eventbus.(*bus).dispatch.func1(0x14000291970?)
	/Users/ilmars/Dev/private/gitagrip/internal/eventbus/eventbus.go:146 +0x50
created by gitagrip/internal/eventbus.(*bus).dispatch in goroutine 36
	/Users/ilmars/Dev/private/gitagrip/internal/eventbus/eventbus.go:140 +0x210

goroutine 43 [select]:
github.com/charmbracelet/bubbletea.(*standardRenderer).listen(0x1400019c0e0)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/standard_renderer.go:149 +0x64
created by github.com/charmbracelet/bubbletea.(*standardRenderer).start in goroutine 1
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/standard_renderer.go:99 +0xb4

goroutine 45 [syscall]:
syscall.syscall6(0x0?, 0x0?, 0x14000185c88?, 0x1049f1cd4?, 0x0?, 0x0?, 0x0?)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/runtime/sys_darwin.go:60 +0x50
golang.org/x/sys/unix.kevent(0x0?, 0x400000?, 0x14000185d08?, 0x1049f5f90?, 0x104d9c7c0?, 0x1049ee8bc?)
	/Users/ilmars/go/pkg/mod/golang.org/x/sys@v0.34.0/unix/zsyscall_darwin_arm64.go:275 +0x54
golang.org/x/sys/unix.Kevent(0x14000185d38?, {0x140001824c8?, 0x14000185d28?, 0x1049ee9d4?}, {0x14000185d58?, 0x104e545f0?, 0x14000185d48?}, 0x104ac3ff4?)
	/Users/ilmars/go/pkg/mod/golang.org/x/sys@v0.34.0/unix/syscall_bsd.go:397 +0x40
github.com/muesli/cancelreader.(*kqueueCancelReader).wait(0x14000182480)
	/Users/ilmars/go/pkg/mod/github.com/muesli/cancelreader@v0.2.2/cancelreader_bsd.go:125 +0x58
github.com/muesli/cancelreader.(*kqueueCancelReader).Read(0x14000182480, {0x140000ac000, 0x100, 0x100})
	/Users/ilmars/go/pkg/mod/github.com/muesli/cancelreader@v0.2.2/cancelreader_bsd.go:69 +0x40
github.com/charmbracelet/bubbletea.readAnsiInputs({0x104bfae58, 0x14000154190}, 0x14000110310, {0x104eebfc8, 0x14000182480})
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/key.go:565 +0x80
github.com/charmbracelet/bubbletea.readInputs(...)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/key_other.go:12
github.com/charmbracelet/bubbletea.(*Program).readLoop(0x140001563c0)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tty.go:99 +0x8c
created by github.com/charmbracelet/bubbletea.(*Program).initCancelReader in goroutine 1
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tty.go:91 +0x180

goroutine 47 [select]:
github.com/charmbracelet/bubbletea.(*Program).listenForResize(0x140001563c0, 0x14000110620)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/signals_unix.go:25 +0xe4
created by github.com/charmbracelet/bubbletea.(*Program).handleResize in goroutine 1
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:323 +0xcc

goroutine 48 [select]:
github.com/charmbracelet/bubbletea.(*Program).handleCommands.func1()
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:340 +0xac
created by github.com/charmbracelet/bubbletea.(*Program).handleCommands in goroutine 1
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:336 +0x94

goroutine 5 [select]:
github.com/charmbracelet/bubbletea.(*Program).Send(...)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:745
github.com/charmbracelet/bubbletea.(*Program).handleCommands.func1.1()
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:365 +0xd0
created by github.com/charmbracelet/bubbletea.(*Program).handleCommands.func1 in goroutine 48
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:354 +0x10c

goroutine 3 [select]:
github.com/charmbracelet/bubbletea.(*Program).Send(...)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:745
github.com/charmbracelet/bubbletea.(*Program).handleCommands.func1.1()
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:365 +0xd0
created by github.com/charmbracelet/bubbletea.(*Program).handleCommands.func1 in goroutine 48
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:354 +0x10c

goroutine 129 [chan receive]:
github.com/charmbracelet/bubbletea.Tick.func1()
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/commands.go:157 +0x3c
github.com/charmbracelet/bubbletea.(*Program).handleCommands.func1.1()
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:364 +0x64
created by github.com/charmbracelet/bubbletea.(*Program).handleCommands.func1 in goroutine 48
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:354 +0x10c

goroutine 113 [syscall]:
syscall.syscall6(0x1000a20c0?, 0x12bad39f8?, 0x104e54a78?, 0x90?, 0x1400005d808?, 0x1400017e120?, 0x140001a3968?)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/runtime/sys_darwin.go:60 +0x50
syscall.wait4(0x140001a3998?, 0x104a9b144?, 0x90?, 0x104bedae0?)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/syscall/zsyscall_darwin_arm64.go:44 +0x4c
syscall.Wait4(0x140000a0380?, 0x140001a39cc, 0x140000a20c0?, 0x140000a0310?)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/syscall/syscall_bsd.go:144 +0x28
os.(*Process).pidWait.func1(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec_unix.go:68
os.ignoringEINTR2[...](...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/file_posix.go:261
os.(*Process).pidWait(0x140004b6100)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec_unix.go:67 +0xa4
os.(*Process).wait(0x3?)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec_unix.go:30 +0x24
os.(*Process).Wait(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec.go:358
os/exec.(*Cmd).Wait(0x1400015a600)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:922 +0x38
os/exec.(*Cmd).Run(0x1400015a600)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:626 +0x38
os/exec.(*Cmd).Output(0x1400015a600)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:1018 +0xbc
gitagrip/internal/git.(*gitService).getStashCount(0x140000b5dd8?, {0x104bfaec8?, 0x140000a0070?}, {0x14000246090, 0x22})
	/Users/ilmars/Dev/private/gitagrip/internal/git/gitservice.go:456 +0xb4
gitagrip/internal/git.(*gitService).RefreshRepo(0x14000124810, {0x104bfaec8, 0x140000a0070}, {0x14000246090, 0x22})
	/Users/ilmars/Dev/private/gitagrip/internal/git/gitservice.go:231 +0x460
gitagrip/internal/git.NewGitService.func2.1()
	/Users/ilmars/Dev/private/gitagrip/internal/git/gitservice.go:84 +0x170
created by gitagrip/internal/git.NewGitService.func2 in goroutine 20
	/Users/ilmars/Dev/private/gitagrip/internal/git/gitservice.go:70 +0x21c

goroutine 28 [select]:
github.com/charmbracelet/bubbletea.(*Program).Send(...)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:745
created by github.com/charmbracelet/bubbletea.(*Program).eventLoop in goroutine 1
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:478 +0x830

goroutine 120 [IO wait]:
internal/poll.runtime_pollWait(0x104f37cc8, 0x72)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/runtime/netpoll.go:351 +0xa0
internal/poll.(*pollDesc).wait(0x1400007c5a0?, 0x140001e4200?, 0x1)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/internal/poll/fd_poll_runtime.go:84 +0x28
internal/poll.(*pollDesc).waitRead(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Read(0x1400007c5a0, {0x140001e4200, 0x200, 0x200})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/internal/poll/fd_unix.go:165 +0x1fc
os.(*File).read(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/file_posix.go:29
os.(*File).Read(0x14000508098, {0x140001e4200?, 0x1400021e558?, 0x104a93ec8?})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/file.go:124 +0x6c
bytes.(*Buffer).ReadFrom(0x14000125530, {0x104bf9da0, 0x1400019a060})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/bytes/buffer.go:211 +0x90
io.copyBuffer({0x104bf9f40, 0x14000125530}, {0x104bf9da0, 0x1400019a060}, {0x0, 0x0, 0x0})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/io/io.go:415 +0x14c
io.Copy(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/io/io.go:388
os.genericWriteTo(0x1?, {0x104bf9f40, 0x14000125530})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/file.go:275 +0x58
os.(*File).WriteTo(0x104d6af00?, {0x104bf9f40?, 0x14000125530?})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/file.go:253 +0x60
io.copyBuffer({0x104bf9f40, 0x14000125530}, {0x104bf9cc0, 0x14000508098}, {0x0, 0x0, 0x0})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/io/io.go:411 +0x98
io.Copy(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/io/io.go:388
os/exec.(*Cmd).writerDescriptor.func1()
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:596 +0x44
os/exec.(*Cmd).Start.func2(0x140002912c0?)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:749 +0x34
created by os/exec.(*Cmd).Start in goroutine 113
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:748 +0x76c

goroutine 121 [IO wait]:
internal/poll.runtime_pollWait(0x104f37a98, 0x72)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/runtime/netpoll.go:351 +0xa0
internal/poll.(*pollDesc).wait(0x1400007c660?, 0x14000388000?, 0x1)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/internal/poll/fd_poll_runtime.go:84 +0x28
internal/poll.(*pollDesc).waitRead(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Read(0x1400007c660, {0x14000388000, 0x8000, 0x8000})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/internal/poll/fd_unix.go:165 +0x1fc
os.(*File).read(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/file_posix.go:29
os.(*File).Read(0x140005080b0, {0x14000388000?, 0x14000054da8?, 0x1049ebca8?})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/file.go:124 +0x6c
io.copyBuffer({0x104bfa2e0, 0x140001540f0}, {0x104bf9da0, 0x1400011f438}, {0x0, 0x0, 0x0})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/io/io.go:429 +0x18c
io.Copy(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/io/io.go:388
os.genericWriteTo(0x1?, {0x104bfa2e0, 0x140001540f0})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/file.go:275 +0x58
os.(*File).WriteTo(0x104d6af00?, {0x104bfa2e0?, 0x140001540f0?})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/file.go:253 +0x60
io.copyBuffer({0x104bfa2e0, 0x140001540f0}, {0x104bf9cc0, 0x140005080b0}, {0x0, 0x0, 0x0})
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/io/io.go:411 +0x98
io.Copy(...)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/io/io.go:388
os/exec.(*Cmd).writerDescriptor.func1()
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:596 +0x44
os/exec.(*Cmd).Start.func2(0x0?)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:749 +0x34
created by os/exec.(*Cmd).Start in goroutine 113
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:748 +0x76c

goroutine 122 [runnable]:
os/exec.(*Cmd).watchCtx(0x1400015a600, 0x14000118150)
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:789 +0x78
created by os/exec.(*Cmd).Start in goroutine 113
	/nix/store/rgmygksbfyy75iappxpinv0y8lxfq35x-go-1.24.6/share/go/src/os/exec/exec.go:775 +0x738

goroutine 67 [select]:
github.com/charmbracelet/bubbletea.(*Program).Send(...)
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:745
github.com/charmbracelet/bubbletea.(*Program).handleCommands.func1.1()
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:365 +0xd0
created by github.com/charmbracelet/bubbletea.(*Program).handleCommands.func1 in goroutine 48
	/Users/ilmars/go/pkg/mod/github.com/charmbracelet/bubbletea@v1.3.7/tea.go:354 +0x10c
