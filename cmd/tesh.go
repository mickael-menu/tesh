package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mickael-menu/tesh/pkg/tesh"
)

func main() {
	if len(os.Args) < 2 {
		exit("missing input folder")
	}
	var err error
	wd := ""
	if len(os.Args) >= 3 {
		wd, err = filepath.Abs(os.Args[2])
		exitIfErr(err)
	}

	suite, err := tesh.ParseSuite(os.Args[1])
	exitIfErr(err)
	report, err := tesh.RunSuite(suite, tesh.RunConfig{
		WorkingDir: wd,
		Callbacks: tesh.RunCallbacks{
			OnFinishCommand: func(test tesh.TestNode, cmd tesh.CommandNode, wd string, err error) {
				if err != nil {
					fmt.Printf("%s:\n%s%s\n\n", test.Name, cmd.DumpShort(), err)
				}
			},
		},
	})
	exitIfErr(err)
	if report.FailedCount == 0 {
		fmt.Printf("PASSED %d tests\n", report.TotalCount)
	} else {
		fmt.Printf("FAILED %d/%d tests\n", report.FailedCount, report.TotalCount)
		os.Exit(1)
	}
}

func exitIfErr(err error) {
	if err != nil {
		exit(err.Error())
	}
}

func exit(msg string) {
	fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	os.Exit(1)
}
