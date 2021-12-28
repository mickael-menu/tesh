package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		exit("missing input file argument")
	}

	data, err := ioutil.ReadFile(os.Args[1])
	exitIfErr(err)
	script, err := ParseScript(string(data))
	exitIfErr(err)
	err = Run(script, RunCallbacks{})
	exitIfErr(err)
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
