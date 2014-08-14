limitnet [![GoDoc](https://godoc.org/gopkg.in/kornel661/limitnet.v0?status.svg)](https://godoc.org/gopkg.in/kornel661/limitnet.v0)
========

Package limitnet provides some network primitives, most notably throttled listener, needed by my [nserv](https://github.com/kornel661/nserv) package.

ThrottledListener implements
* throttling the number of active connections (preventing depletion of server's resources and DOS attacks) and
* graceful shutdown through the Wait method.


Usage
=====

```go
import "gopkg.in/kornel661/limitnet.v0"
```
or
```
go get gopkg.in/kornel661/limitnet.v0
```