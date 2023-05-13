// goFrundis
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"

	"codeberg.org/anaseto/gofrundis/exporter/latex"
	"codeberg.org/anaseto/gofrundis/exporter/markdown"
	"codeberg.org/anaseto/gofrundis/exporter/mom"
	"codeberg.org/anaseto/gofrundis/exporter/tpl"
	"codeberg.org/anaseto/gofrundis/exporter/xhtml"
	"codeberg.org/anaseto/gofrundis/frundis"
)

func main() {

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	optFormat := flag.String("T", "", "export `format` (required)")
	optAllInOneFile := flag.Bool("a", false, "all in one file (for xhtml only)")
	optStandalone := flag.Bool("s", false, "standalone document (default for xhtml and epub)")
	optOutputFile := flag.String("o", "", "`output-file`")
	optCompress := flag.Bool("z", false, "produce a finalized compressed EPUB (zipped)")
	optTemplate := flag.Bool("t", false, "template operation mode")
	optExec := flag.Bool("x", false, "unrestricted mode (#run and shell filters allowed)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -T format [-a] [-s] [-t] [-x] [-o output-file] path\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "See man page frundis(1) for details.")
	}
	flag.Parse()

	if *cpuprofile != "" {
		// profiling
		f, err := os.Create(*cpuprofile)
		if err != nil {
			Error(false, err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	args := flag.Args()
	var filename string
	if len(args) > 0 {
		filename = args[0]
	} else {
		Error(true, "filename required")
	}
	if len(args) > 1 {
		Error(true, "too many arguments")
	}

	switch *optFormat {
	case "epub", "xhtml", "latex", "markdown", "mom":
	case "":
		Error(true, "-T option required")
	default:
		Error(true, "invalid format argument to -T option")
	}
	if *optOutputFile == "" {
		if *optFormat == "epub" || *optFormat == "xhtml" && !*optAllInOneFile {
			Error(true, "-o option required with formats epub and xhtml (without -a)")
		}
	}

	if *optTemplate {
		export(
			tpl.NewExporter(&tpl.Options{
				OutputFile: *optOutputFile,
				Format:     *optFormat}),
			filename,
			*optExec)
		os.Exit(0)
	}

	switch *optFormat {
	case "epub", "xhtml":
		export(
			xhtml.NewExporter(&xhtml.Options{
				Format:       *optFormat,
				OutputFile:   *optOutputFile,
				Standalone:   *optStandalone,
				AllInOneFile: *optAllInOneFile}),
			filename,
			*optExec)
		if *optFormat == "epub" && *optCompress {
			err := writeEpub(*optOutputFile, *optOutputFile+".epub")
			if err != nil {
				fmt.Fprintf(os.Stderr, "frundis: %v", err)
				os.Exit(1)
			}
		}
	case "latex":
		export(
			latex.NewExporter(&latex.Options{
				OutputFile: *optOutputFile,
				Standalone: *optStandalone}),
			filename,
			*optExec)
	case "markdown":
		export(
			markdown.NewExporter(&markdown.Options{OutputFile: *optOutputFile}),
			filename,
			*optExec)
	case "mom":
		export(
			mom.NewExporter(&mom.Options{
				OutputFile: *optOutputFile,
				Standalone: *optStandalone}),
			filename,
			*optExec)
	}
}

func export(exp frundis.Exporter, filename string, unrestricted bool) {
	err := frundis.ProcessFrundisSource(exp, filename, unrestricted)
	if err != nil {
		Error(false, err)
	}
}

func Error(usage bool, msgs ...interface{}) {
	s := "frundis: "
	s += fmt.Sprint(msgs...)
	fmt.Fprintln(os.Stderr, s)
	if usage {
		flag.Usage()
	}
	os.Exit(1)
}

func Log(format string, msgs ...interface{}) {
	fmt.Fprintf(os.Stderr, "frundis: "+format, msgs...)
}
