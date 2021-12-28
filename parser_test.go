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

func TestParseCommand(t *testing.T) {
	testParse(t, "$  echo 'hello world' ", []Stmt{
		CommandStmt{Cmd: "echo 'hello world'"},
	})
}

func TestParseEmptyCommandErrorsOut(t *testing.T) {
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

func testParse(t *testing.T, content string, expected []Stmt) {
	actual, err := Parse(content)
	assert.Nil(t, err)
	assert.Equal(t, actual, expected)
}

func testParseErr(t *testing.T, content string, msg string) {
	_, err := Parse(content)
	assert.Err(t, err, msg)
}
