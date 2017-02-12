// goFrundis
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/anaseto/gofrundis/exporter/latex"
	"github.com/anaseto/gofrundis/exporter/markdown"
	"github.com/anaseto/gofrundis/exporter/mom"
	"github.com/anaseto/gofrundis/exporter/tpl"
	"github.com/anaseto/gofrundis/exporter/xhtml"
	"github.com/anaseto/gofrundis/frundis"
)

func main() {

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	optFormat := flag.String("T", "", "export `format` (required)")
	optAllInOneFile := flag.Bool("a", false, "all in one file (for xhtml only)")
	optStandalone := flag.Bool("s", false, "standalone document (default for xhtml and epub)")
	optOutputFile := flag.String("o", "", "`output-file`")
	optTemplate := flag.Bool("t", false, "template operation mode")
	optExec := flag.Bool("x", false, "unrestricted mode (#run and shell filters allowed)")
	flag.Parse()

	if *cpuprofile != "" {
		// profiling
		f, err := os.Create(*cpuprofile)
		if err != nil {
			Error(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	args := flag.Args()
	var filename string
	if len(args) > 0 {
		filename = args[0]
	} else {
		Error("filename required")
	}
	if len(args) > 1 {
		Error("too many arguments")
	}

	switch *optFormat {
	case "epub", "xhtml", "latex", "markdown", "mom":
	case "":
		Error("-T option required")
	default:
		Error("invalid format argument to -T option")
	}
	if *optOutputFile == "" {
		if *optFormat == "epub" || *optFormat == "xhtml" && !*optAllInOneFile {
			Error("-o option required with formats epub and xhtml (without -a)")
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
		Error(err)
	}
}

func Error(msgs ...interface{}) {
	s := "frundis:"
	s += fmt.Sprint(msgs...)
	fmt.Fprintln(os.Stderr, s)
	os.Exit(1)
}
