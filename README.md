Frundis
-------

The *frundis* language is a semantic markup language with a simplified
roff-like syntax. It was originally intended for supporting authoring of
novels, but it can be used for more varied documents.  It relies on the
exporting capabilities of the tool `frundis` to LaTeX, XHTML, EPUB (2 and 3),
markdown and groff mom.

The language has a focus on simplicity. It provides a few flexible built-in
macros with extensible semantics. It strives to provide good error messages and
catch typos, while still allowing one to finely control output for a specific
format when needed.

Here is a list of its main features:

+ Common elements such as links, images, cross-references, lists, simple
  tables, table of contents …
+ Arbitrary metadata for EPUB. Indexed html files.
+ User defined markup tags with configurable rendering.
+ Raw blocks, file inclusion, filters, macros and variables.
+ Roff-like syntax: simple, clear and friendly to grep and diff.

Documentation
-------------

The tool and language are explained in detail in manpages doc/frundis.1 and
doc/frundis\_syntax.5.

The [website of the project](https://frundis.tuxfamily.org/) serves an html
version of the manual pages, as well as a FAQ, examples and other relevant
information.

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
