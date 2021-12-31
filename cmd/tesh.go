package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mickael-menu/tesh/pkg/tesh"
)

func main() {
	var err error

	var update bool
	flag.BoolVar(&update, "u", false, "overwrite test cases instead of failing")
	var printBytes bool
	flag.BoolVar(&printBytes, "b", false, "print bytes instead of strings")
	flag.Parse()

	values := flag.Args()

	if len(values) == 0 {
		fmt.Println("usage: tesh [-u] <tests> [<working-dir>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	testsDir := values[0]
	wd := ""
	if len(values) >= 2 {
		wd, err = filepath.Abs(values[1])
		exitIfErr(err)
	}

	suite, err := tesh.ParseSuite(testsDir)
	exitIfErr(err)
	report, err := tesh.RunSuite(suite, tesh.RunConfig{
		Update:     update,
		WorkingDir: wd,
		Callbacks: tesh.RunCallbacks{
			OnFinishCommand: func(test tesh.TestNode, cmd tesh.CommandNode, config tesh.RunConfig, err error) {
				if err != nil {
					fmt.Printf("FAIL %s: $ %s\n", test.Name, cmd.Cmd)
					switch err := err.(type) {
					case tesh.ExitCodeAssertError:
						fmt.Printf("\t%s\n", err)
					case tesh.DataAssertError:
						fmt.Printf("expected on %s:\n---\n", err.FD.String())
						if printBytes {
							fmt.Println([]byte(err.Expected))
						}
						fmt.Println(err.Expected)
						fmt.Println("---\ngot:\n---")
						if printBytes {
							fmt.Println([]byte(err.Received))
						}
						fmt.Println(err.Received)
						fmt.Println("---")
					}
				} else {
					fmt.Printf("OK %s: $ %s\n", test.Name, cmd.Cmd)
				}
			},
		},
	})
	exitIfErr(err)
	if update && report.UpdatedCount > 0 {
		fmt.Printf("UPDATED %d on %d tests\n", report.UpdatedCount, report.TotalCount)
	} else if report.FailedCount == 0 {
		fmt.Printf("PASSED %d tests\n", report.TotalCount)
	} else {
		fmt.Printf("FAILED %d on %d tests\n", report.FailedCount, report.TotalCount)
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
