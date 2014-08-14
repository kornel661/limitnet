package limitnet

import (
	"net"
	"sync"
)

// throttledConn - a connection that gives a token back when closed
type throttledConn struct {
	net.Conn
	once         sync.Once
	replaceToken func()
}

func (tc *throttledConn) Close() error {
	err := tc.Conn.Close()
	tc.once.Do(tc.replaceToken) // do it only once (even when sb closes a conn multiple times)
	return err
}
