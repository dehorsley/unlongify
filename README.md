Unlongify
=========

A stupid little tool to port legacy C to x86/x86-64. 

Installation
------------

If you have [Go](https://golang.org) installed, and `$GOPATH/bin` in your PATH, 
just run

    go get github.com/dehorsley/unlongify

Usage
-----


    unlongify <PATH>

Eg:

    unlongify /usr2/st

This recursively scans a directory tree for C source files and headers
and modifies them to change `long` type declarations to `int`. Care is
taken to avoid false positives elsewhere in the source. Printf/scanf
format options are also updated to use `int`.

**WARNING:** this does not make backups before editing files.


Why does this exist?
--------------------

GCC on x86 processors compiles both `int` and `long` to 32-bit integers, whereas
on x86\_64 `int` compiles to 32-bit and `long` compiles to 64-bit. Certain older
code bases mix these two, mostly as they were written in the 16 bit era.
This tool removes some of the grunt work in correcting this, but doesn't really
understand C, and should be treated at a blunt object to get the code close to
correct.

Users writing modern code should consider updating any program
interfaces to use fixed width integers defined in `stdint.h`. This is
more portable between compilers and architectures. This tool can be
modified to do some of this work for you too.

Note: not all longs are bad, specifically some system calls explicitly
require and return `long` arguments. For example `mtype` field in the
struct argument to `msgrcv` must be of type `long`. **This tool will
blindly convert these to `int`s**! You'll need to go back and
un-unlongify these.

Users should check their code after using this tool. Modern versions of
GCC will warn in at least some cases check the compilers output.
