package limitnet

import (
	"errors"
	"net"
	"os"
)

// Filer interface is satisfied by those listeners that allow getting duplicates
// of their file descriptors. Both *net.TCPListener and *net.UnixListener satisfy
// Filer.
type Filer interface {
	File() (f *os.File, err error)
}

// CopyFD returns a duplicate (dup) of a file descriptor associated with l.
func CopyFD(l net.Listener) (fd uintptr, err error) {
	if list, ok := l.(Filer); ok {
		var file *os.File
		if file, err = list.File(); err == nil {
			return file.Fd(), nil
		}
		return 0, err
	}
	if list, ok := l.(*throttledListener); ok {
		return CopyFD(list.Listener)
	}
	return 0, errors.New("Cannot get a dup of fd from the listener.")
}
