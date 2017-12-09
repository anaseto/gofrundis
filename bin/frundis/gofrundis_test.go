package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"testing"

	"github.com/anaseto/gofrundis/exporter/latex"
	"github.com/anaseto/gofrundis/exporter/markdown"
	"github.com/anaseto/gofrundis/exporter/mom"
	"github.com/anaseto/gofrundis/exporter/tpl"
	"github.com/anaseto/gofrundis/exporter/xhtml"
	"github.com/anaseto/gofrundis/frundis"
)

func TestMain(m *testing.M) {
	err := os.Setenv("FRUNDIS", "ok")
	if err != nil {
		fmt.Fprint(os.Stderr, err)
	}
	err = os.Chdir("../../testdata")
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	err = os.Setenv("FRUNDISLIB", "data/includes")
	if err != nil {
		fmt.Fprint(os.Stderr, err)
	}
	os.Exit(m.Run())
}

func TestWarnings(t *testing.T) {
	doWarnings(t, "warnings.frundis", "xhtml")
}

func TestWarningsNbsp(t *testing.T) {
	doWarnings(t, "warnings-nbsp.frundis", "xhtml")
}

func TestWarningsEpub(t *testing.T) {
	doWarnings(t, "warnings-epub.frundis", "epub")
}

func doWarnings(t *testing.T, path, format string) {
	err := os.RemoveAll(".gofrundis_warnings_test")
	if err != nil {
		t.Fatal(err)
	}
	exp := xhtml.NewExporter(
		&xhtml.Options{
			Format:     format,
			OutputFile: ".gofrundis_warnings_test",
			Werror:     ioutil.Discard,
			//Werror:       os.Stderr,
			AllInOneFile: true})
	err = frundis.ProcessFrundisSource(exp, path, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFragments(t *testing.T) {
	dataDir, err := os.Open("data")
	if err != nil {
		t.Fatalf("Error reading data: %v", err)
	}
	defer func() {
		err := dataDir.Close()
		if err != nil {
			t.Fatalf("Error closing data: %v", err)
		}
	}()
	names, err := dataDir.Readdirnames(-1)
	if err != nil {
		t.Logf("Could not read directory: %v", err)
	}
	for _, f := range names {
		if b, _ := path.Match("*.frundis", f); !b {
			continue
		}
		fullPath := path.Join("data", f)
		for _, format := range []string{"latex", "mom", "xhtml", "markdown"} {
			t.Run(fullPath+"-"+format, func(t *testing.T) {
				doFile(t, fullPath, format, false)
			})
		}
	}
	t.Run("tpl.frundis", func(t *testing.T) {
		doFile(t, "tpl.frundis", "xhtml", true)
	})
}

func TestStandalones(t *testing.T) {
	dataDir, err := os.Open("data-dirs")
	if err != nil {
		t.Fatalf("Error reading data-dirs: %v", err)
	}
	defer func() {
		err = dataDir.Close()
		if err != nil {
			t.Fatalf("Error closing data-dirs: %v", err)
		}
	}()
	names, err := dataDir.Readdirnames(-1)
	if err != nil {
		t.Logf("Could not read directory: %v", err)
	}
	for _, f := range names {
		if b, _ := path.Match("*.frundis", f); !b {
			continue
		}
		fullPath := path.Join("data-dirs", f)
		if b, _ := path.Match("*-epub*", f); b {
			t.Run(fullPath+" "+"*-epub", func(t *testing.T) {
				doStandalone(t, fullPath, "epub", false)
			})
			continue
		}
		if b, _ := path.Match("*-xhtml*", f); b {
			t.Run(fullPath+" "+"*-xhtml", func(t *testing.T) {
				doStandalone(t, fullPath, "xhtml", false)
			})
			t.Run(fullPath+" "+"*-xhtml", func(t *testing.T) {
				doStandalone(t, fullPath, "xhtml", true)
			})
			continue
		}
		if b, _ := path.Match("*-latex*", f); b {
			t.Run(fullPath+" "+"*-latex", func(t *testing.T) {
				doStandalone(t, fullPath, "latex", true)
			})
			continue
		}
		t.Run(fullPath+" "+"xhtml", func(t *testing.T) {
			doStandalone(t, fullPath, "xhtml", false)
		})
		for _, format := range []string{"xhtml", "latex", "mom"} {
			t.Run(fullPath+" "+format, func(t *testing.T) {
				doStandalone(t, fullPath, format, true)
			})
		}
	}
}

var outputFile = ".gofrundistest.out"
var outputDir = ".gofrundistestdir.out"

func doFile(t *testing.T, file string, format string, tplmode bool) {
	name := strings.TrimSuffix(file, ".frundis")
	suffix := strings.Replace(format, "xhtml", "html", -1)
	suffix = strings.Replace(suffix, "latex", "tex", -1)
	var exp frundis.Exporter
	switch format {
	case "xhtml":
		if tplmode {
			exp = tpl.NewExporter(&tpl.Options{
				OutputFile: outputFile,
				Format:     format})
		} else {
			exp = xhtml.NewExporter(
				&xhtml.Options{
					Format:       "xhtml",
					OutputFile:   outputFile,
					AllInOneFile: true})
		}
	case "latex":
		exp = latex.NewExporter(&latex.Options{OutputFile: outputFile})
	case "markdown":
		exp = markdown.NewExporter(&markdown.Options{OutputFile: outputFile})
	case "mom":
		exp = mom.NewExporter(&mom.Options{OutputFile: outputFile})
	}
	err := frundis.ProcessFrundisSource(exp, file, true)
	ref := name + "." + suffix
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("diff", "-u", ref, outputFile)
	_, e := os.Stat(ref)
	var diff []byte
	if e != nil {
		diff = []byte("Reference file does not exist yet.\n")
	} else {
		var err error
		diff, err = cmd.CombinedOutput()
		if err != nil {
			switch err := err.(type) {
			case *exec.ExitError:
				s := err.Sys().(syscall.WaitStatus)
				if s.ExitStatus() >= 2 {
					t.Fatalf("Error executing command: %s: %s", strings.Join(cmd.Args, " "), string(diff))
				}
			default:
				t.Fatal(err)
			}
		}
	}
	if !(string(diff) == "") {
		t.Error(string(diff))
	}
}

func doStandalone(t *testing.T, file string, format string, toFile bool) {
	var suffix string
	switch format {
	case "epub":
		suffix = "-epub"
	case "xhtml":
		if toFile {
			suffix = ".html"
		} else {
			suffix = "-html"
		}
	case "latex":
		suffix = ".tex"
	case "mom":
		suffix = ".mom"
	default:
		t.Fatalf("internal error:unknown format: %s", format)
	}
	name := strings.TrimSuffix(file, ".frundis")
	info, err := os.Stat(outputDir)
	if err == nil {
		if info.IsDir() {
			err = os.RemoveAll(outputDir)
			if err != nil {
				t.Fatalf("removing outputDir: %v", err)
			}
		} else {
			os.Remove(outputDir)
		}
	}
	var exp frundis.Exporter
	switch format {
	case "xhtml", "epub":
		exp = xhtml.NewExporter(
			&xhtml.Options{
				Format:       format,
				OutputFile:   outputDir,
				Standalone:   true,
				AllInOneFile: toFile})
	case "latex":
		exp = latex.NewExporter(
			&latex.Options{
				OutputFile: outputDir,
				Standalone: true})
	case "markdown":
		exp = markdown.NewExporter(&markdown.Options{OutputFile: outputFile})
	case "mom":
		exp = mom.NewExporter(
			&mom.Options{
				OutputFile: outputDir,
				Standalone: true})
	}
	err = frundis.ProcessFrundisSource(exp, file, false)
	if err != nil {
		t.Fatal(err)
	}
	ref := name + suffix
	_, e := os.Stat(ref)
	var diff []byte
	if e != nil {
		diff = []byte("Reference file does not exist yet\n")
	} else {
		cmd := exec.Command("diff", "-ru", ref, outputDir)
		var err error
		diff, err = cmd.CombinedOutput()
		if err != nil {
			switch err := err.(type) {
			case *exec.ExitError:
				s := err.Sys().(syscall.WaitStatus)
				if s.ExitStatus() >= 2 {
					t.Error("^^^^^^^ DIFF ERROR ^^^^^^^^^^^^^^^^^")
					t.Error(string(diff))
					t.Error("^^^^^^^ END OF ERROR ^^^^^^^^^")
					t.FailNow()
				}
			default:
				t.Fatal(err)
			}
		}
	}
	if !(string(diff) == "") {
		t.Error(string(diff))
	}
}
