package main

import (
	"testing"

	"github.com/mickael-menu/tesh/internal/util/test/assert"
)

func TestRunEmpty(t *testing.T) {
	testRun(t, ``)
}

func TestRunSuccess(t *testing.T) {
	testRun(t, `
# Test output on stdout
$ echo "hello\nworld"
>hello
>world

# Test output on stderr
1$ cat not-found
2>cat: not-found: No such file or directory

# Test input from stdin
$ cat -n
<Testing input
<on several lines
>     1	Testing input
>     2	on several lines

# Test exit code
42$ exit 42
`)
}

func TestRunFailureStdout(t *testing.T) {
	testRunErr(t, `
$ echo "hello"
>world
`,
		DataAssertError{
			FD:       Stdout,
			Expected: "world\n",
			Received: "hello\n",
		},
	)
}

func TestRunFailureStderr(t *testing.T) {
	testRunErr(t, `
$ echo "hello" 1>&2
2>world
`,
		DataAssertError{
			FD:       Stderr,
			Expected: "world\n",
			Received: "hello\n",
		},
	)
}

func TestRunFailureExitCode(t *testing.T) {
	testRunErr(t, "$ exit 24",
		ExitCodeAssertError{
			Expected: 0,
			Received: 24,
		},
	)
	testRunErr(t, "24$ exit 0",
		ExitCodeAssertError{
			Expected: 24,
			Received: 0,
		},
	)
}

func testRun(t *testing.T, content string) {
	script, err := ParseScript(content)
	assert.Nil(t, err)
	err = Run(script, RunCallbacks{})
	assert.Nil(t, err)
}

func testRunErr(t *testing.T, content string, expected error) {
	script, err := ParseScript(content)
	assert.Nil(t, err)
	err = Run(script, RunCallbacks{})
	assert.Equal(t, err, expected)
}
