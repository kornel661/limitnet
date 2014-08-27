package limitnet

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	// FDsFlagName is the name of the commandline flag used to pass information
	// about fds to the child process. The flag has the form:
	//     -flag=n-m
	// what means that file descriptors n, n+1, ..., m-1 are the 'inherited'
	// listeners.
	FDsFlagName = "netlimit.FDs"

	fdsFlag = flag.String(FDsFlagName, "", "internal limitnet flag")
)

// Filer interface is satisfied by those listeners that allow getting duplicates
// of their file descriptors. Both *net.TCPListener and *net.UnixListener satisfy
// Filer.
type Filer interface {
	File() (f *os.File, err error)
}

// CopyFD returns a duplicate (dup) of a file descriptor associated with l.
func CopyFD(l net.Listener) (fd *os.File, err error) {
	if list, ok := l.(Filer); ok {
		var file *os.File
		if file, err = list.File(); err == nil {
			return file, nil
		}
		return nil, err
	}
	if list, ok := l.(*throttledListener); ok {
		return CopyFD(list.Listener)
	}
	return nil, errors.New("Cannot get a dup of fd from the listener.")
}

// PrepareCmd returns a *os/exec.Cmd ready to be run. The executed program will
// inherit listeners ls. Use RetriveListeners in the executed program to get them.
//
// Name is the name of the command to be run (leave empty for this program). Args
// are the arguments passed and ls the listeners to be 'inherited'.
//
// You can fine-tune fields of cmd but changing Args or ExtraFiles might break
// your 'listener inheritance' setup.
func PrepareCmd(name string, args []string, extraFiles []*os.File, ls ...net.Listener) (cmd *exec.Cmd, err error) {
	if name == "" {
		this, err := filepath.Abs(os.Args[0])
		if err != nil {
			return nil, err
		}
		name = this
	}
	files := make([]*os.File, len(ls))
	for i, l := range ls {
		files[i], err = CopyFD(l)
		if err != nil {
			return nil, err
		}
	}

	start := 3 + len(extraFiles)
	flag := fmt.Sprintf("-%s=%d-%d", FDsFlagName, start, start+len(files))
	extraFiles = append(extraFiles, files...)

	args = append([]string{flag}, args...)
	cmd = exec.Command(name, args...)
	//cmd.Stdin = os.Stdin
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	cmd.ExtraFiles = extraFiles
	return cmd, nil
}

func readFDsFlag() (a, b uintptr, err error) {
	if *fdsFlag == "" {
		return 0, 0, errors.New("Program run without inheriting listeners.")
	}
	if n, err := fmt.Sscanf(*fdsFlag, "%d-%d", &a, &b); err != nil || n != 2 {
		if err == nil {
			err = errors.New("Error parsing netlimitFDs flag.")
		} else {
			err = errors.New("Error parsing netlimitFDs flag: " + err.Error())
		}
		return 0, 0, err
	}
	return
}

// CanRetrieveListeners returns true if it seems there are listeners to be
// inherited.
func CanRetrieveListeners() bool {
	if _, _, err := readFDsFlag(); err != nil {
		return false
	}
	return true
}

// RetrieveListeners if invoked in a process created by PrepareCmd(...).Start()
// returns the list of 'inherited' listeners.
//
// flag.Parse() needs to be invoked first.
func RetrieveListeners() (ls []net.Listener, err error) {
	var a, b uintptr
	if a, b, err = readFDsFlag(); err != nil {
		return nil, err
	}
	if b < a {
		return nil, errors.New("Invalid values for netlimitFDs flag.")
	}
	ls = make([]net.Listener, b-a)
	for fd, i := a, 0; fd < b; {
		file := os.NewFile(fd, "listener")
		ls[i], err = net.FileListener(file)
		file.Close()
		if err != nil {
			return nil, err
		}
		fd++
		i++
	}
	return ls, err
}
