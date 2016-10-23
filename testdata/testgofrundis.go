package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

func main() {
	dataDir, err := os.Open("t/data")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading t/data:%v\n", err)
		os.Exit(1)
	}
	names, err := dataDir.Readdirnames(-1)
	for _, f := range names {
		if b, _ := path.Match("*.frundis", f); !b {
			continue
		}
		fullPath := path.Join("t", "data", f)
		testFile(fullPath, "latex")
		testFile(fullPath, "xhtml")
		testFile(fullPath, "markdown")
	}
	err = dataDir.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error closing t/data:%v\n", err)
		os.Exit(1)
	}
	dataDir, err = os.Open("t/data-dirs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading t/data-dirs:%v\n", err)
		os.Exit(1)
	}
	names, err = dataDir.Readdirnames(-1)
	for _, f := range names {
		if b, _ := path.Match("*.frundis", f); !b {
			continue
		}
		fullPath := path.Join("t", "data-dirs", f)
		if b, _ := path.Match("*-epub*", f); b {
			testStandalone(fullPath, "epub", false)
			continue
		}
		if b, _ := path.Match("*-xhtml*", f); b {
			testStandalone(fullPath, "xhtml", false)
			testStandalone(fullPath, "xhtml", true)
			continue
		}
		if b, _ := path.Match("*-latex*", f); b {
			testStandalone(fullPath, "latex", true)
			continue
		}
		testStandalone(fullPath, "xhtml", false)
		testStandalone(fullPath, "xhtml", true)
		testStandalone(fullPath, "latex", true)
	}
	err = dataDir.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error closing t/data-dirs:%v\n", err)
		os.Exit(1)
	}
}

var outputFile = ".gofrundistest.out"
var outputDir = ".gofrundistestdir.out"

func getBinPath() string {
	binPath, okEnv := os.LookupEnv("GOPATH")
	if !okEnv {
		fmt.Fprintf(os.Stderr, "no GOPATH")
		os.Exit(1)
	}
	binPath = path.Join(binPath, "bin", "frundis")
	return binPath
}

func testFile(file string, format string) {
	binPath := getBinPath()
	name := strings.TrimSuffix(file, ".frundis")
	suffix := strings.Replace(format, "xhtml", "html", -1)
	suffix = strings.Replace(suffix, "latex", "tex", -1)
	cmd := exec.Command(binPath, "-T", format, "-a", "-o", outputFile, file)
	cmdout, err := cmd.CombinedOutput()
	fmt.Fprint(os.Stderr, string(cmdout))
	ref := name + "." + suffix
	if err != nil {
		ok(false, ref)
		fmt.Fprint(os.Stderr, err)
		return
	}
	cmd = exec.Command("diff", "-u", ref, outputFile)
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
				ok(false, ref)
				if s.ExitStatus() >= 2 {
					fmt.Fprintf(os.Stderr, "Error executing command:%s:%s", strings.Join(cmd.Args, " "), string(diff))
					return
				}
			default:
				ok(false, ref)
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
	if !ok(string(diff) == "", ref) {
		fmt.Fprint(os.Stderr, string(diff))
		input := readLine()
		if input == "Y" {
			b, err := ioutil.ReadFile(outputFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading:%s", outputFile)
				return
			}
			err = ioutil.WriteFile(ref, b, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing:%s", ref)
				return
			}

		}
	}
}

func testStandalone(file string, format string, toFile bool) {
	binPath := getBinPath()
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
	default:
		fmt.Fprintf(os.Stderr, "internal error:unknown format:%s", format)
		os.Exit(1)
	}
	name := strings.TrimSuffix(file, ".frundis")
	info, err := os.Stat(outputDir)
	if err == nil {
		if info.IsDir() {
			err = os.RemoveAll(outputDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "removing outputDir:%v", err)
				os.Exit(1)
			}
		} else {
			os.Remove(outputDir)
		}
	}
	var cmd *exec.Cmd
	if toFile {
		cmd = exec.Command(binPath, "-T", format, "-a", "-s", "-o", outputDir, file)
	} else {
		cmd = exec.Command(binPath, "-T", format, "-o", outputDir, file)
	}
	cmdout, err := cmd.CombinedOutput()
	fmt.Fprint(os.Stderr, string(cmdout))
	ref := name + suffix
	cmdExpression := strings.Join(cmd.Args, " ")
	if err != nil {
		ok(false, ref)
		fmt.Fprintf(os.Stderr, "Error executing command:%s\n", cmdExpression)
		fmt.Fprint(os.Stderr, err)
		return
	}
	_, e := os.Stat(ref)
	var diff []byte
	if e != nil {
		diff = []byte("Reference file does not exist yet\n")
	} else {
		cmd = exec.Command("diff", "-ru", ref, outputDir)
		var err error
		diff, err = cmd.CombinedOutput()
		if err != nil {
			switch err := err.(type) {
			case *exec.ExitError:
				s := err.Sys().(syscall.WaitStatus)
				if s.ExitStatus() >= 2 {
					ok(false, ref)
					fmt.Fprintf(os.Stderr, "Error for command:%s\n", cmdExpression)
					fmt.Fprintf(os.Stderr, "Error executing command:%s\n", strings.Join(cmd.Args, " "))
					fmt.Fprint(os.Stderr, string(diff))
					fmt.Fprintln(os.Stderr, "^^^^^^^ END OF ERROR ^^^^^^^^^")
					return
				}
			default:
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
	if !ok(string(diff) == "", ref) {
		fmt.Fprintf(os.Stderr, "Diff for command:%s\n", cmdExpression)
		fmt.Fprint(os.Stderr, string(diff))
		input := readLine()
		if input == "Y" {
			fmt.Fprintf(os.Stderr, "replacing %s with %s\n", ref, outputFile)
			err := os.RemoveAll(ref)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error removing directory %s:%v", ref, err)
				os.Exit(1)
			}
			err = os.Rename(outputDir, ref)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error renaming to %s", ref)
				os.Exit(1)
			}
		}
	}
}

func readLine() string {
	fmt.Fprintf(os.Stderr, "Files differ. Put new [Y/n]?")
	in := bufio.NewScanner(os.Stdin)
	okScan := in.Scan()
	if !okScan {
		return ""
	}
	return in.Text()
}

var testNum int

func ok(b bool, msg string) bool {
	testNum++
	if b {
		fmt.Fprintf(os.Stderr, "ok %d - %s\n", testNum, msg)
	} else {
		fmt.Fprintf(os.Stderr, "not ok %d - %s\n", testNum, msg)
	}
	return b
}
