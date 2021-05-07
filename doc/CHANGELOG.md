This changelog file only lists important changes.

## v0.10

+ New xhtml-chap-custom-filenames, xhtml-chap-prefix and xhtml-custom ids
  for better control over file names in multi XHTML and EPUB documents, and
  for facilitating stable URLs.
+ New -alt option for Im macro, for providing alternate text in XHTML and
  EPUB outputs.
+ Replaced xhtml-xelatex parameter with an xhtml-variant parameter, for
  better future extensibility.

## v0.9

Mainly a bugfix release, with also some typographic improvements in
french documents.

## v0.8

+ New -eq option for #if macro. Mainly a convenience that allows to use a
  single #dv flag variable instead of several when more than two cases are to
  be considered.

## v0.7

+ Add support for reference by header number in cross-references.
+ Minor breaking change: when mixing $1, $2, etc. and $@, the latter gets only
  remaining arguments.
+ Fix french non-breaking space handling in -b and -e mtag options.
+ Fix bug in handling of deeply nested macros.
+ Fix bug in ".Ef -ns"
+ Improvements and updates in man page.

## v0.6

+ minor markdown export fix for paragraph title
+ minor documentation fixes	

## v0.5

+ interpolate environment variables with \*[$ENVVAR]
+ some bug fixes

## v0.4

+ warn if unterminated quoted argument instead of just adding closing
  automatically
+ warn if invalid -c argument to html mtag or dtag
+ add some other warnings
+ update code.frundis example

## v0.3

+ cross-reference support overhaul (much simpler now)
+ some minor fixes

## v0.2

+ support attributes for dtags too
+ better external command support
+ disallow external commands by default
+ some minor fixes and refactorings

## v0.1 (changes with respect to the perl version)

+ no -perl flag in .#de
+ no #fl
+ .Bd -t literal now to be done with "escape" and "verbatim" filters
+ \$@ can be used to interpolate all arguments of an user defined macro
+ -filter option should now be spelled -shell
+ Ft makes use of -t option too
+ markdown exporter (not polished)
+ groff mom exporter (not polished)
+ added experimental restricted mode more template-friendly
+ added named arguments and flags to user defined macros
+ added simple substitution filters and regexp filters
+ added attribute support
+ many minor fixes and improvements