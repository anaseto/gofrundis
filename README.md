Frundis
-------

*frundis* is a tool for compiling documents written in the [frundis
language](https://frundis.tuxfamily.org/man/frundis_syntax-5.html), a semantic
markup language primarily intended for supporting authoring of novels, but also
well suited for many other kinds of documents. The [frundis
tool](https://frundis.tuxfamily.org/man/frundis-1.html) can export documents
into to LaTeX, XHTML 5, EPUB, markdown and groff mom.

The language has a focus on simplicity. It provides a few flexible built-in
macros with extensible semantics. It strives to provide good error messages and
catch typos, while still allowing one to finely control output for a specific
format when needed.

Here is a list of its main features:

+ Common elements such as links, images, cross-references, lists, simple
  tables, table of contents …
+ Arbitrary metadata for EPUB. Indexed HTML files.
+ User defined markup tags with configurable rendering.
+ Raw blocks, file inclusion, filters, conditionals, macros and variables.
+ Roff-like syntax: simple, clear and friendly to grep and diff.

Documentation
-------------

Both the tool and language are explained in detail in the [website of the
project](https://frundis.tuxfamily.org/). The website serves an html version of
the manual pages, as well as a FAQ, examples and other relevant information.

Install
-------

+ Install the [go compiler](https://golang.org/).
+ Add `$(go env GOPATH)/bin` to your `$PATH` (for example `export PATH="$PATH:$(go env GOPATH)/bin"`).
+ Run the command `go install ./cmd/frundis`.
  
The `frundis` command should now be available.

No dependencies outside of the go standard library.

Editor Support
--------------

There is a frundis-specific vim syntax file under doc/vim/. Others editors
should do fine by using any built-in general mode for roff/nroff files.
