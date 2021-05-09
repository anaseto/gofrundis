Description
-----------

The *frundis* language is a semantic markup language with a simplified
roff-like syntax. It was originally intended for supporting authoring of
novels, but it can be used for more varied documents.  It relies on the
exporting capabilities of the tool `frundis` to LaTeX, XHTML, EPUB (2 and 3),
markdown and groff mom.  *Only LaTeX, XHTML and EPUB output formats are
considered complete and mature.*

The language focuses on simplicity, using a few flexible built-in macros with
sensible defaults, and it strives to provide good error messages, while
allowing one to explicitly mess up when needed and finely control output for a
specific format.

Here is a list of its main features:

+ Common elements such as links, images, cross-references, lists, simple
  tables, table of contents …
+ Arbitrary metadata for EPUB. Indexed html files.
+ User defined markup tags with configurable rendering.
+ Raw blocks, file inclusion, filters, macros and variables.
+ Roff-like syntax: simple, clear and friendly to grep and diff.

The tool and language are explained in detail in manpages doc/frundis.1 and
doc/frundis\_syntax.5. They are available too in html form in the [frundis
website](https://frundis.tuxfamily.org/).

You can have a look at examples in frundis\_syntax(5) man page in the EXAMPLES
section, as well as in files in the doc/examples directory.

Install
-------

+ Install the [go compiler](https://golang.org/).
+ Add `$(go env GOPATH)/bin` to your `$PATH` (for example `export PATH="$PATH:$(go env GOPATH)/bin"`).
+ Run the command `go install` in the `cmd/frundis/` directory.
  
The `frundis` command should now be available.

No dependencies outside of the go standard library.

Editor Support
--------------

There is a frundis-specific vim syntax file under doc/vim/. Others editors
should do fine by using any built-in general mode for roff/nroff files.
