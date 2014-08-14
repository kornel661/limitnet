package limitnet

import (
	"log"
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

func (c *counter) Disconnect() {
	c.Lock()
	defer c.Unlock()
	c.curr--
}

func throttleConn(delay time.Duration, t *testing.T, count *counter) {
	conn, err := net.Dial("tcp", addr)
	//log.Println("connected")
	//defer log.Println("disconnected")
	if err != nil {
		t.Fatal(err)
		return
	}
	count.Connect()
	time.Sleep(delay)
	count.Disconnect()
	if err := conn.Close(); err != nil {
		t.Error(err)
	}
}

func TestThrottling(t *testing.T) {
	max := 10
	count := &counter{Limit: max, t: t, connected: make(chan struct{}, 2*max+1)}
	tl := newTL(t)
	tl.MaxConns(max)
	log.Println("Starting connections")
	for i := 0; i < 10*max; i++ {
		go throttleConn(10*time.Millisecond, t, count)
	}
	log.Println("Waiting for all to connect")
	for i := 0; i < 10*max; i++ {
		<-count.connected // wait for all goroutines to connect
	}
	log.Println("Closing listener")
	tl.Close()
	tl.Wait()
}
