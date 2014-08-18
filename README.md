limitnet
========

Package limitnet provides some network primitives, most notably throttled listener, needed by my [nserv](https://github.com/kornel661/nserv) package.

ThrottledListener implements
* throttling the number of active connections (preventing depletion of server's resources and DOS attacks) and
* graceful shutdown through the Wait method.


Usage
=====

```go
import "gopkg.in/kornel661/limitnet.v1"
```
or
```
go get gopkg.in/kornel661/limitnet.v1
```


Versions
========

* Developement version (v0)
  [![GoDoc](https://godoc.org/gopkg.in/kornel661/limitnet.v0?status.svg)](https://godoc.org/gopkg.in/kornel661/limitnet.v0) [![GoWalker](https://gowalker.org/api/v1/badge)](https://gowalker.org/gopkg.in/kornel661/limitnet.v0) [![GoCover](http://gocover.io/_badge/gopkg.in/kornel661/limitnet.v0)](http://gocover.io/gopkg.in/kornel661/limitnet.v0)


Changelog
=========

* 2014.08.16 (version v0): Testing & bug hunting season opened.
