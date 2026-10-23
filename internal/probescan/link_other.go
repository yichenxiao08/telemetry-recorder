//go:build !windows

package probescan

import "os"

func unsafeLink(i os.FileInfo) bool { return i.Mode()&os.ModeSymlink != 0 }
