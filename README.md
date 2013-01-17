# makefs

Makefs is a virtual filesystem processor for go.

The idea is to take a
[http.FileSystem](http://golang.org/pkg/net/http/#FileSystem) and apply
[make](http://en.wikipedia.org/wiki/Make_(software\))-style rules to it,
producing a new virtual `http.FileSystem` as the output.

The resulting file system can be served using the
[http.FileServer](http://golang.org/pkg/net/http/#FileServer). Just like with
make, recipes will only be re-executed if your sources (prerequisites) have
changed, making it a very efficient / pleasent to work with.

Alternatively, makefs also offers you to write the resulting file system to
disk (which is useful for static site generators), or into a `.go` file, which
you can statically link into your application for deployment.

## Introduction

@TODO

## License

@TODO
