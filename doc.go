/*
Package limitnet provides some network primitives, most notably throttled
listener, needed by my nserv (https://gopkg.in/kornel661/nserv.v0) package.

Features:
* throttled listener
* helper functions for writing a zero-downtime server (by passing an open fd to
  a child process, see nserv package for examples)


Usage:

	import "gopkg.in/kornel661/limitnet.v0"

or

	go get gopkg.in/kornel661/limitnet.v0

Replace v0 by the version you need (v0 is a development version, with no API
stability guaratees).

For up-to-date changelog and features list see [README]
(https://github.com/kornel661/limitnet/blob/master/README.md).
*/
package limitnet
