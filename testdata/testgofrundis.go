package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

var passTests = true

func main() {
	err := os.Setenv("FRUNDIS", "ok")
	if err != nil {
		fmt.Fprint(os.Stderr, "could not set environment variable")
	}
	err = os.Setenv("FRUNDISLIB", "data/includes")
	if err != nil {
		fmt.Fprint(os.Stderr, err)
	}
	for _, f := range []func() error{doFragments, doStandalones} {
		err := f()
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	}
	if !passTests {
		os.Exit(1)
	}
}

func doFragments() error {
	dataDir, err := os.Open("data")
	if err != nil {
		return fmt.Errorf("Error reading data:%v", err)
	}
	defer func() {
		err := dataDir.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error closing data:%v", err)
		}
	}()
	names, err := dataDir.Readdirnames(-1)
	for _, f := range names {
		if b, _ := path.Match("*.frundis", f); !b {
			continue
		}
		fullPath := path.Join("data", f)
		for _, format := range []string{"latex", "mom", "xhtml", "markdown"} {
			err := doFile(fullPath, format, false)
			if err != nil {
				return err
			}
		}
	}
	doFile("tpl.frundis", "xhtml", true)
	return nil
}

func doStandalones() error {
	dataDir, err := os.Open("data-dirs")
	if err != nil {
		return fmt.Errorf("Error reading data-dirs:%v", err)
	}
	defer func() {
		err = dataDir.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error closing data-dirs:%v", err)
		}
	}()
	names, err := dataDir.Readdirnames(-1)
	for _, f := range names {
		if b, _ := path.Match("*.frundis", f); !b {
			continue
		}
		fullPath := path.Join("data-dirs", f)
		if b, _ := path.Match("*-epub*", f); b {
			err := doStandalone(fullPath, "epub", false)
			if err != nil {
				return err
			}
			continue
		}
		if b, _ := path.Match("*-xhtml*", f); b {
			err := doStandalone(fullPath, "xhtml", false)
			if err != nil {
				return err
			}
			err = doStandalone(fullPath, "xhtml", true)
			if err != nil {
				return err
			}
			continue
		}
		if b, _ := path.Match("*-latex*", f); b {
			err := doStandalone(fullPath, "latex", true)
			if err != nil {
				return err
			}
			continue
		}
		err := doStandalone(fullPath, "xhtml", false)
		if err != nil {
			return err
		}
		for _, format := range []string{"xhtml", "latex", "mom"} {
			err = doStandalone(fullPath, format, true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

var outputFile = ".gofrundistest.out"
var outputDir = ".gofrundistestdir.out"

func getBinPath() (string, error) {
	binPath, okEnv := os.LookupEnv("GOPATH")
	if !okEnv {
		return "", errors.New("no GOPATH")
	}
	binPath = path.Join(binPath, "bin", "frundis")
	return binPath, nil
}

func doFile(file string, format string, tpl bool) error {
	binPath, err := getBinPath()
	if err != nil {
		return err
	}
	name := strings.TrimSuffix(file, ".frundis")
	suffix := strings.Replace(format, "xhtml", "html", -1)
	suffix = strings.Replace(suffix, "latex", "tex", -1)
	var cmd *exec.Cmd
	if tpl {
		cmd = exec.Command(binPath, "-T", format, "-t", "-o", outputFile, file)
	} else {
		cmd = exec.Command(binPath, "-T", format, "-x", "-a", "-o", outputFile, file)
	}
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	ref := name + "." + suffix
	if err != nil {
		ok(false, ref)
		fmt.Fprint(os.Stderr, err)
		return nil
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
					return nil
				}
			default:
				ok(false, ref)
				return err
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
				return nil
			}
			err = ioutil.WriteFile(ref, b, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing:%s", ref)
				return nil
			}

		}
	}
	return nil
}

func doStandalone(file string, format string, toFile bool) error {
	binPath, err := getBinPath()
	if err != nil {
		return err
	}
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
		return fmt.Errorf("internal error:unknown format:%s", format)
	}
	name := strings.TrimSuffix(file, ".frundis")
	info, err := os.Stat(outputDir)
	if err == nil {
		if info.IsDir() {
			err = os.RemoveAll(outputDir)
			if err != nil {
				return fmt.Errorf("removing outputDir:%v", err)
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
	_, err = cmd.CombinedOutput()
	ref := name + suffix
	cmdExpression := strings.Join(cmd.Args, " ")
	if err != nil {
		ok(false, ref)
		fmt.Fprintf(os.Stderr, "Error executing command:%s\n", cmdExpression)
		fmt.Fprint(os.Stderr, err)
		return nil
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
					return nil
				}
			default:
				return err
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
				return fmt.Errorf("Error removing directory %s:%v", ref, err)
			}
			err = os.Rename(outputDir, ref)
			if err != nil {
				return fmt.Errorf("Error renaming to %s", ref)
			}
		}
	}
	return nil
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
		passTests = false
	}
	return b
}
