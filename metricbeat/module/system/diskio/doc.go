/*
Package diskio fetches disk IO metrics from the OS. It is implemented for
darwin (requires cgo), freebsd, linux, and windows.

Detailed descriptions of IO stats provided by Linux can be found here:
https://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/plain/Documentation/iostats.txt?id=refs/tags/v4.6-rc7
*/
package diskio

//go:generate go run run.go -cmd "go tool cgo -godefs defs_diskio_windows.go" -goarch amd64 -output defs_diskio_windows_amd64.go
//go:generate go run run.go -cmd "go tool cgo -godefs defs_diskio_windows.go" -goarch 386 -output defs_diskio_windows_386.go
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zdiskio_windows.go diskio_windows.go
//go:generate gofmt -w defs_diskio_windows_amd64.go defs_diskio_windows_386.go zdiskio_windows.go
