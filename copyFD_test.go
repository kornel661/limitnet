package limitnet

import (
	"net"
	"testing"
)

func TestCopyFD(t *testing.T) {
	var l net.Listener = newTL(t)
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
