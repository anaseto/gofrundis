This changelog file only lists important changes.

## v0.13.0 2022-10-26

+ Improved support for english typography.
+ New -ns option in Em macro.
+ Added html coloration script in misc/ repository.

## v0.12.0 2021-05-10

+ Now lists and tables can handle more general content in HTML, EPUB and LaTeX
  exports.
+ The Lk macro can now take several arguments for the text, instead of just
  one.
+ HTML 5 and EPUB 3 are now the default (minor incompatible change).

## v0.11.0 2021-05-08

+ Added option to produce a compressed EPUB out of the box, without requiring
  manually zipping the directory.
+ Add the “viewport” meta header when producing HTML5.

## v0.10.0 2021-05-07

+ New xhtml-chap-custom-filenames, xhtml-chap-prefix and xhtml-custom ids
  for better control over file names in multi XHTML and EPUB documents, and
  for facilitating stable URLs.
+ New -alt option for Im macro, for providing alternate text in XHTML and
  EPUB outputs.
+ Replaced xhtml-xelatex parameter with an xhtml-variant parameter, for
  better future extensibility (minor incompatible change).

## v0.9.0 2020-11-15

Mainly a bugfix release, with also some typographic improvements in
french documents.

## v0.8 2019-11-09

+ New -eq option for #if macro. Mainly a convenience that allows to use a
  single #dv flag variable instead of several when more than two cases are to
  be considered.

## v0.7 2019-08-26

+ Add support for reference by header number in cross-references.
+ Minor breaking change: when mixing $1, $2, etc. and $@, the latter gets only
  remaining arguments.
+ Fix french non-breaking space handling in -b and -e mtag options.
+ Fix bug in handling of deeply nested macros.
+ Fix bug in ".Ef -ns"
+ Improvements and updates in man page.

## v0.6 2017-07-01

+ minor markdown export fix for paragraph title
+ minor documentation fixes	

## v0.5 2017-06-03

+ interpolate environment variables with \*[$ENVVAR]
+ some bug fixes

## v0.4 2017-03-12

+ warn if unterminated quoted argument instead of just adding closing
  automatically
+ warn if invalid -c argument to html mtag or dtag
+ add some other warnings
+ update code.frundis example

## v0.3 2017-02-19

+ cross-reference support overhaul (much simpler now)
+ some minor fixes

## v0.2 2017-02-12

+ support attributes for dtags too
+ better external command support
+ disallow external commands by default
+ some minor fixes and refactorings

## v0.1 2017-02-05

These are changes with respect to the perl version:

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
