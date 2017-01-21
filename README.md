Description
-----------

The *frundis* language intends to be a semantic markup language with a
simplified roff-like syntax, originally intended for supporting authoring of
novels, but it can be used for more varied stuff.  It relies on the exporting
capabilities of the tool `frundis` to LaTeX, XHTML, EPUB (2 and 3),
markdown and groff mom. Only LaTeX, XHTML and EPUB output formats are
considered mature.

The language focuses on simplicity, using a few flexible built-in macros, and
strives to provide good error messages, while allowing one to explicitly mess
up when needed and finely control output for a specific format.

The tool and language are explained in detail in manpages doc/frundis.1 and
doc/frundis_syntax.5. They are available too in html form in the [frundis
website](http://bardinflor.perso.aquilenet.fr/frundis/intro-en).

You can have a look at examples in frundis_syntax(5) man page in the EXAMPLES
section, as well as in files in the doc/examples directory.

Install
-------

+ Install the [go compiler](https://golang.org/).
+ Set `$GOPATH` variable (for example `export GOPATH=$HOME/go`).
+ Add `$GOPATH/bin` to your `$PATH`.

Then use the command `go get github.com/anaseto/gofrundis/bin/frundis`.
  
The `frundis` command should now be available.

No dependencies outside of the go standard library.

Editor Support
--------------

There is a frundis-specific vim syntax file under doc/vim/. Others editors
should do fine by using any built-in general mode for roff/nroff files.
