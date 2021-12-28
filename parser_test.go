package main

import (
	"testing"

	"github.com/mickael-menu/tesh/internal/util/test/assert"
)

func TestParseEmpty(t *testing.T) {
	testParse(t, "   \n  \n", []Stmt{})
}

func TestParseComment(t *testing.T) {
	testParse(t, "# Comment on one line", []Stmt{
		CommentStmt{Content: "Comment on one line"},
	})
}

func TestParseEmptyComment(t *testing.T) {
	testParse(t, "#", []Stmt{
		CommentStmt{Content: ""},
	})
}

func TestParseMultilineComment(t *testing.T) {
	testParse(t, "# Comment written \n   #on several\n#   lines ", []Stmt{
		CommentStmt{Content: "Comment written\non several\nlines"},
	})
}

func TestParseInlineComment(t *testing.T) {
	testParseErr(t, " prefix # comment", "a comment must start on its own line, line 1")
}

func TestParseCommand(t *testing.T) {
	testParse(t, "$  echo 'hello world' ", []Stmt{
		CommandStmt{Cmd: "echo 'hello world'"},
	})
}

func TestParseEmptyCommand(t *testing.T) {
	testParseErr(t, "$   ", "unexpected empty command, line 1")
}

func TestParseMultipleCommands(t *testing.T) {
	testParse(t, `
$  echo "hello world"
  $zk list --link-to test.md
`, []Stmt{
		CommandStmt{Cmd: `echo "hello world"`},
		CommandStmt{Cmd: `zk list --link-to test.md`},
	})
}

func TestParseCommandInvalidPrefix(t *testing.T) {
	testParseErr(t, " pref$  cmd", "invalid command prefix: `pref`")
}

func TestParseStdin(t *testing.T) {
	testParse(t, " <  Input sent to the program ", []Stmt{
		DataStmt{FD: Stdin, Content: "  Input sent to the program "},
	})
}

func TestParseEmptyStdin(t *testing.T) {
	testParse(t, "	<", []Stmt{
		DataStmt{FD: Stdin, Content: ""},
	})
}

func TestParseMultipleStdin(t *testing.T) {
	testParse(t, `
	<  Input sent to a program 
<	
< which spans
<several lines    

<Another one
`, []Stmt{
		DataStmt{FD: Stdin, Content: "  Input sent to a program \n	\n which spans\nseveral lines    "},
		DataStmt{FD: Stdin, Content: "Another one"},
	})
}

func TestParseStdinInvalidPrefix(t *testing.T) {
	testParseErr(t, " pref<  Error", "invalid data prefix: `pref`")
}

func TestParseStdout(t *testing.T) {
	testParse(t, " >  Output from a program ", []Stmt{
		DataStmt{FD: Stdout, Content: "  Output from a program "},
	})
}

func TestParseEmptyStdout(t *testing.T) {
	testParse(t, "	>", []Stmt{
		DataStmt{FD: Stdout, Content: ""},
	})
}

func TestParseMultipleStdout(t *testing.T) {
	testParse(t, `
	>  Output from a program 
>	
> which spans
>several lines    

>Another one
`, []Stmt{
		DataStmt{FD: Stdout, Content: "  Output from a program \n	\n which spans\nseveral lines    "},
		DataStmt{FD: Stdout, Content: "Another one"},
	})
}

func TestParseStdoutInvalidPrefix(t *testing.T) {
	testParseErr(t, " a>  Error", "invalid data prefix: `a`")
}

func TestParseStderr(t *testing.T) {
	testParse(t, " 2>  Error from a program ", []Stmt{
		DataStmt{FD: Stderr, Content: "  Error from a program "},
	})
}

func TestParseEmptyStderr(t *testing.T) {
	testParse(t, "	2>", []Stmt{
		DataStmt{FD: Stderr, Content: ""},
	})
}

func TestParseMultipleStderr(t *testing.T) {
	testParse(t, `
	2>  Error from a program 
2>	
2> which spans
2>several lines    

2>Another one
`, []Stmt{
		DataStmt{FD: Stderr, Content: "  Error from a program \n	\n which spans\nseveral lines    "},
		DataStmt{FD: Stderr, Content: "Another one"},
	})
}

func TestParseCompleteExample(t *testing.T) {
	testParse(t, `
# Create a file
$ echo "hello" > test

# Read the created file
$ cat test
>hello

# Read a file that doesn't exist
$ cat unknown
2>cat: unknown: No such file or directory

$ read input
<Input content

$ ls
>total 8
>-rw-r--r--  1 mickael     6B Dec 28 10:16 test
`, []Stmt{
		CommentStmt{Content: "Create a file"},
		CommandStmt{Cmd: `echo "hello" > test`},
		CommentStmt{Content: "Read the created file"},
		CommandStmt{Cmd: "cat test"},
		DataStmt{FD: Stdout, Content: "hello"},
		CommentStmt{Content: "Read a file that doesn't exist"},
		CommandStmt{Cmd: "cat unknown"},
		DataStmt{FD: Stderr, Content: "cat: unknown: No such file or directory"},
		CommandStmt{Cmd: "read input"},
		DataStmt{FD: Stdin, Content: "Input content"},
		CommandStmt{Cmd: "ls"},
		DataStmt{FD: Stdout, Content: "total 8\n-rw-r--r--  1 mickael     6B Dec 28 10:16 test"},
	})
}

func testParse(t *testing.T, content string, expected []Stmt) {
	actual, err := Parse(content)
	assert.Nil(t, err)
	assert.Equal(t, actual, expected)
}

func testParseErr(t *testing.T, content string, msg string) {
	_, err := Parse(content)
	assert.Err(t, err, msg)
}
