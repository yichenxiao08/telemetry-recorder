//go:build windows

package probescan

import (
	"os"
	"syscall"
)

func unsafeLink(i os.FileInfo) bool {
	a, ok := i.Sys().(*syscall.Win32FileAttributeData)
	return i.Mode()&os.ModeSymlink != 0 || ok && a.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
