package limitnet

import (
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// counter counts no of all accepted connections (all), currently active connections
// (curr), maximum number of simultaneously active connections (max) and fails the
// the test if Limit is exceeded.
// The channel connected records connected connections.
type counter struct {
	// Limit of simultaneous connections.
	Limit int
	// ErrFun is a functions to be invoked in case of too many simultaneous connections.
	ErrFun         func()
	curr, max, all int
	sync.Mutex
}

func (c *counter) Connect() {
	c.Lock()
	defer c.Unlock()
	c.all++
	c.curr++
	if c.curr > c.max {
		c.max = c.curr
	}
	if c.curr > c.Limit {
		c.ErrFun()
	}
}

func (c *counter) Disconnect() {
	c.Lock()
	defer c.Unlock()
	c.curr--
}

func (c *counter) All() int {
	c.Lock()
	defer c.Unlock()
	return c.all
}

func (c *counter) Max() int {
	c.Lock()
	defer c.Unlock()
	return c.max
}

func (c *counter) Curr() int {
	c.Lock()
	defer c.Unlock()
	return c.curr
}

// throttleConn - "client" goroutine.
func throttleConn(delay time.Duration, t *testing.T, connected chan<- struct{}) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Error(err)
		return
	}

	data := []byte{0}
	conn.Read(data)

	if err := conn.Close(); err != nil {
		t.Error(err)
	} else {
		connected <- struct{}{}
	}
}

// server goroutine.
func server(tl *throttledListener, delay time.Duration, t *testing.T, cntr *counter) {
	for {
		conn, err := tl.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") { // closing the listener signals the server to stop
				t.Errorf("Error. %s", err)
			}
			return
		}
		// service connection
		go func() {
			cntr.Connect()
			time.Sleep(delay) // simulate work
			cntr.Disconnect()
			if err := conn.Close(); err != nil {
				t.Error(err)
			}
		}()
	}
}

func TestThrottling(t *testing.T) {
	max := 10
	cntr := &counter{Limit: max}
	cntr.ErrFun = func() {
		t.Errorf("Error. Exceeded number of simultaneous connections: %d, limit: %d.", cntr.curr, cntr.Limit)
	}
	tl := newTL(t)
	connDelay := 50 * time.Millisecond
	go server(tl, connDelay, t, cntr)
	tl.MaxConns(max)
	delay()

	// connected counts successful connections
	connected := make(chan struct{}, 10*max)
	for i := 0; i < 10*max; i++ {
		go throttleConn(5*connDelay, t, connected)
	}
	// wait for all goroutines to connect
	for i := 0; i < 10*max; i++ {
		<-connected
	}

	tl.Close()
	currClose := cntr.Curr()
	tl.Wait()
	currWait := cntr.Curr()
	if currWait > 0 {
		t.Errorf("Error. Still %d connections active after Wait().", currWait)
	}
	t.Logf("Connections active after close: %d, after wait: %d", currClose, currWait)
	if all := cntr.All(); all != 10*max {
		t.Errorf("Error. Only %d connections accepted out of %d", all, 10*max)
	}
	t.Logf("Maximum number of concurrent connections: %d, limit: %d", cntr.Max(), max)
	if err := tl.Close(); err != errClosing {
		t.Errorf("Error. Closing the listener twice should give %s instead of %s", errClosing, err)
	}
}
