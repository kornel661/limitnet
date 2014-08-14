package limitnet

import (
	"net"
	"runtime"
	"sync"
	"testing"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

type counter struct {
	Limit          int
	t              fataler
	curr, max, all int
	sync.Mutex
	connected chan struct{}
}

func (c *counter) Connect() {
	c.connected <- struct{}{}
	c.Lock()
	defer c.Unlock()
	c.all++
	c.curr++
	if c.curr > c.Limit {
		c.t.Fatalf("connections: %d, limit: %d", c.curr, c.Limit)
	}
	if c.curr > c.max {
		c.max = c.curr
	}
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

func (c *counter) Disconnect() {
	c.Lock()
	defer c.Unlock()
	c.curr--
}

func throttleConn(delay time.Duration, t *testing.T, count *counter) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
		return
	}
	time.Sleep(delay) // doesn't really matter
	if err := conn.Close(); err != nil {
		t.Error(err)
	}
}

func server(tl *throttledListener, delay time.Duration, t *testing.T, count *counter) {
	for {
		conn, err := tl.Accept()
		if err != nil {
			if err.Error() != errClosing.Error() { // closing the listener signals the server to stop
				t.Fatal(err)
			}
			return
		}
		// service connection
		go func() {
			count.Connect()
			time.Sleep(delay)
			count.Disconnect()
			if err := conn.Close(); err != nil {
				t.Error(err)
			}
		}()
	}
}

func TestThrottling(t *testing.T) {
	max := 10
	count := &counter{Limit: max, t: t, connected: make(chan struct{}, 2*max+1)}
	tl := newTL(t)
	connDelay := 50 * time.Millisecond
	go server(tl, connDelay, t, count)
	tl.MaxConns(max)
	for i := 0; i < 10*max; i++ {
		go throttleConn(5*connDelay, t, count)
	}
	for i := 0; i < 10*max; i++ {
		<-count.connected // wait for all goroutines to connect
	}
	tl.Close()
	currClose := count.Curr()
	tl.Wait()
	currWait := count.Curr()
	if currWait > 0 {
		t.Errorf("Still %d connections active after Wait().", currWait)
	}
	t.Logf("Connections active after close: %d, after wait: %d", currClose, currWait)
	if all := count.All(); all != 10*max {
		t.Errorf("Only %d connections accepted out of %d", all, 10*max)
	}
	t.Logf("Maximum number of concurrent connections: %d, limit: %d", count.Max(), max)
	if err := tl.Close(); err != errClosing {
		t.Errorf("Closing the listener twice shoud give %s instead of %s", errClosing, err)
	}
}
