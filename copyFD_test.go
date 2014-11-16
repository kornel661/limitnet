package limitnet

import (
	"fmt"
	"net"
	"os"
	"testing"
)

func TestCopyFD(t *testing.T) {
	var l = net.Listener(newTL(t))
	defer func() {
		if err := l.Close(); err != nil {
			t.Error(err)
		}
	}()
	if fd, err := CopyFD(l); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Got FD: %v\n", fd)
		if err := fd.Close(); err != nil {
			t.Error(err)
		}
	}
}

func TestRetrieveListeners(t *testing.T) {
	// we don't expect to inherit any listeners
	InitializeZeroDowntime()
	*FDsFlag = ""
	ls, err := RetrieveListeners()
	if err == nil {
		t.Error("The flag parsed successfully.")
	}
	if CanRetrieveListeners() {
		t.Error("The flag parsed successfully.")
	}
	if len(ls) != 0 {
		t.Error("Some listeners inherited.")
	}
	// check if inheriting 0 listeners works:
	*FDsFlag = "3-3"
	if !CanRetrieveListeners() {
		t.Error("Can't retrieve listeners.")
	}
	ls, err = RetrieveListeners()
	if err != nil {
		t.Error(err)
	}
	if ls == nil {
		t.Error("Didn't get the listener list.")
	}
	if len(ls) != 0 {
		t.Errorf("Inherited %d listeners instead of 0.", len(ls))
	}
}

func TestRetrieveOneListener(t *testing.T) {
	// check if inheriting 1 listener works:
	l := newTL(t)
	fd, err := CopyFD(l)
	if err != nil {
		t.Fatal(err)
	}
	err = l.Close()
	if err != nil {
		t.Error(err)
	}

	*FDsFlag = fmt.Sprintf("%d-%d", fd.Fd(), fd.Fd()+1)
	if !CanRetrieveListeners() {
		t.Error("Can't retrieve listeners.")
	}
	ls, err := RetrieveListeners()
	if err != nil {
		t.Fatal(err)
	}
	if ls == nil {
		t.Fatal("Didn't get the listener list.")
	}
	if len(ls) != 1 {
		t.Fatalf("Inherited %d listeners instead of 1.", len(ls))
	}

	err = ls[0].Close()
	if err != nil {
		t.Error(err)
	}
}

func TestPrepareCmd(t *testing.T) {
	name := ""
	args := []string{"arg1", "arg2"}
	extraFiles := []*os.File{{}}
	l := newTL(t)
	cmd, err := PrepareCmd(name, args, extraFiles, l)
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf("%#v", cmd)

	err = l.Close()
	if err != nil {
		t.Error(err)
	}

	if l := len(cmd.ExtraFiles); l != 2 {
		t.Fatalf("Got %d extra files instead of 2.", l)
	}
	err = cmd.ExtraFiles[1].Close()
	if err != nil {
		t.Error(err)
	}
}
