package limitnet

import (
	"log"
	"net"
	"runtime"
	"testing"
	"time"
)

const addr = "localhost:12345"

// delay is a poor man's 'wait for tokens to stabilize'
func delay() {
	runtime.Gosched()
	time.Sleep(20 * time.Millisecond)
}

type fataler interface {
	Fatal(...interface{})
	Fatalf(string, ...interface{})
}

func newTL(t fataler) *throttledListener {
	if l, err := net.Listen("tcp", addr); err != nil {
		t.Fatal(err)
	} else {
		return (NewThrottledListener(l)).(*throttledListener)
	}
	return nil
}

func TestThrottler(t *testing.T) {
	tl := newTL(t)
	max := 200
	// test setting
	tl.MaxConns(max / 2)
	delay()
	if l := tl.MaxConns(-1); l != max/2 {
		t.Errorf("Number of tokens=%d instead of %d. (max/2)", l, max/2)
	}

	for i := 1; i <= 10; i++ {
		tl.MaxConns(max / i)
	}
	for i := 10; i >= 1; i-- {
		tl.MaxConns(max / i)
	}
	for i := 1; i <= 10; i++ {
		tl.MaxConns(max / i)
		runtime.Gosched()
	}
	delay()
	if l := tl.MaxConns(-1); l != max/10 {
		t.Errorf("Number of throtte tokens is %d instead of %d. (after loops)", l, max/10)
	}

	go tl.Close()
	go tl.MaxConns(5) // should work
	log.Println("Waiting for the listener to close...")
	tl.Wait()
}

// BenchmarkThrottler is a benchmark measuring time it takes to add and remove a token.
func BenchmarkThrottler(b *testing.B) {
	tl := newTL(b)
	tl.MaxConns(b.N)
	for len(tl.throttle) != b.N {
		runtime.Gosched() // let the tokens be accumulated
		//log.Printf("tokens: %d\n", len(tl.throttle))
	}
	tl.Close()
	tl.Wait()
}
