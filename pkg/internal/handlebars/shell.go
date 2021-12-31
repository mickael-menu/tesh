package handlebars

import (
	"fmt"
	"os"
	"strings"

	"github.com/aymerick/raymond"
	"github.com/mickael-menu/tesh/pkg/internal/util/exec"
)

func init() {
	// Registers the {{sh}} template helper, which runs shell commands.
	//
	// {{#sh "tr '[a-z]' '[A-Z]'"}}Hello, world!{{/sh}} -> HELLO, WORLD!
	// {{sh "echo 'Hello, world!'"}} -> Hello, world!
	raymond.RegisterHelper("sh", func(arg string, options *raymond.Options) string {
		cmd := exec.CommandFromString(arg)

		// Feed any block content as piped input
		cmd.Stdin = strings.NewReader(options.Fn())

		output, err := cmd.Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "{{sh}} command failed: %v\n", err)
			return ""
		}

		return strings.TrimSpace(string(output))
	})
}
