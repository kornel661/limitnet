package limitnet

import (
	"errors"
	"math"
	"net"
)

var (
	// errClosing, following the convention at https://golang.org/src/pkg/net/net.go?h=OpError#L285
	errClosing = errors.New("use of closed network connection")
	// maxTokens is maximum for MaxConns and the capacity of the throttle channel.
	maxTokens = math.MaxInt32
)

// token defines type used as tokens in channel communication
type token struct{}

// throttledListener (or rather its pointer) implements ThrottledListener interface.
type throttledListener struct {
	net.Listener            // standard net.Listener to wrap
	throttle     chan token // connections take tokens from this channel
	maxThrottle  chan int32 // channel to communicate max number of simultaneously open connections, closing it signals throttler goroutine to stop
	closed       chan token // this channel is closed when listener gets closed
	finished     chan token // this channel is closed when listener is closed and all connections terminated
}

// ThrottledListener type of net.Listener with dynamically adjusted max number
// of simultaneous connections. Needs to be closed to avoid leaking memory.
// Throttling incoming connections is important for any production server as without
// it it's hard to fight off DOS attacks.
//
// Additionally it provides Wait method which waits till all connections accepted
// from the listener are closed. Wait makes it very easy to implement graceful
// shutdown of a server for example.
type ThrottledListener interface {
	// Listener - a standard net.Listener functionality.
	net.Listener
	// Wait returns only when the listener is closed and all connections terminated.
	Wait()
	// MaxConns sets a new maximum limit for simultaneous connections (which can't
	// be greater than math.MaxInt32) and returns the number of free slots for new
	// connections [limit changes are gradual thus the returned value does not
	// necessarily equal (last limit)-(no of active connections) but should quickly
	// attain this value].
	// If n < 0 then the limit is not changed.
	// Can be called after closing the listener.
	//
	// NOTE: It's possible (?) that operating system's kernel starts accepting
	// connections without waiting for the userspace. Anyway, Go will be only able
	// to accept MaxConns connections.
	MaxConns(n int) int
}

// NewThrottledListener returns initialized instance of ThrottledListener wrapping l.
func NewThrottledListener(l net.Listener) ThrottledListener {
	tl := throttledListener{Listener: l}

	tl.throttle = make(chan token, maxTokens)
	tl.maxThrottle = make(chan int32, 1)

	tl.closed = make(chan token, 1)
	tl.closed <- token{}

	tl.finished = make(chan token)

	go tl.throttler()
	return &tl
}

// Accept waits for and returns the next connection to the listener.
func (tl *throttledListener) Accept() (net.Conn, error) {
	if !tl.takeToken() {
		// accepted closed listener, return appropriate error
		var netw = ""
		if _, ok := (tl.Listener).(*net.TCPListener); ok {
			netw = "tcp"
		}
		if _, ok := (tl.Listener).(*net.UnixListener); ok {
			netw = "unix"
		}
		return nil, &net.OpError{Op: "accept", Addr: tl.Addr(), Err: errClosing, Net: netw}
	}
	// now we've got a token
	c, err := tl.Listener.Accept()
	if err != nil { // accept failed, replace token, return error
		tl.replaceToken()
		return nil, err
	}
	// success, return connection that replaces token when closed
	return &throttledConn{Conn: c, replaceToken: tl.replaceToken}, nil
}

// Close closes the listener. Close the listener after use to avoid memory leaks.
func (tl *throttledListener) Close() error {
	_, ok := <-tl.closed // try to take mutex
	if ok {              // the one who'd taken a token closes the channel
		close(tl.closed)      // signal others that we're closing
		close(tl.maxThrottle) // signal throttler goroutine to stop
	} else { // listener's been closed already
		return errClosing
	}
	return tl.Listener.Close()
}

// Wait waits for the listener to be closed and all connections terminated and
// then returns.
func (tl *throttledListener) Wait() {
	<-tl.finished
	return
}

// MaxConns sets a new limit of simultaneously active connections (if n>=0) and
// returns how many more connections can be accepted at the moment.
func (tl *throttledListener) MaxConns(n int) (free int) {
	_, ok := <-tl.closed // lock mutex
	free = len(tl.throttle)
	if !ok { // we're closing
		return
	}
	// unlock mutex:
	defer func() { tl.closed <- token{} }()
	if n < 0 {
		return
	}
	if n > maxTokens {
		n = maxTokens
	}
	tl.maxThrottle <- int32(n)
	return
}

// takeToken takes a token from the 'jar'.
func (tl *throttledListener) takeToken() bool {
	_, ok := <-tl.throttle
	return ok
}

// replaceToken puts a token back to the 'jar'.
func (tl *throttledListener) replaceToken() {
	tl.throttle <- token{}
}

// throttler is run in a separate goroutine. It listens on tl.maxThrottle
// and adds or removes tokens from the tl.throttle channel ('jar').
// Closed tl.maxThrottle channel signals the throttler goroutine to collect all tokens
// (i.e. wait for all connections) end exit.
func (tl *throttledListener) throttler() {
	var (
		instMax   int32  // instantaneous max == number of throttling tokens at large
		targetMax int32  // target instantaneous max, we want to make instMax = targetMax
		ok        = true // if it's OK to continue or should we stop
	)
	// removes a token from the jar
	decrease := func() {
		select {
		case <-tl.throttle:
			instMax--
		case targetMax, ok = <-tl.maxThrottle:
		}
	}
	// adds a token to the jar
	increase := func() {
		select {
		case tl.throttle <- token{}:
			instMax++
		case targetMax, ok = <-tl.maxThrottle:
		}
	}
	// listens for a new instMax (when instMax == targetMax)
	idle := func() {
		select {
		case targetMax, ok = <-tl.maxThrottle:
		}
	}
	// loop until signalled to exit
	for ok {
		switch {
		case instMax < targetMax:
			increase()
		case instMax == targetMax:
			idle()
		case instMax > targetMax:
			decrease()
		}
	}

	// reclaim all tokens (i.e., wait for all connections to finish)
	for i := int32(0); i < instMax; i++ {
		<-tl.throttle
	}
	instMax = 0
	// signal we're finished
	close(tl.throttle)
	close(tl.finished)
}
