package main

import (
	"testing"

	"github.com/mickael-menu/tesh/internal/util/test/assert"
)

func TestParseScriptEmpty(t *testing.T) {
	testParseScript(t, "   ", ScriptNode{})
}

func TestParseScriptComplete(t *testing.T) {
	testParseScript(t, `# Script header

# Create a file
$ echo "hello\nworld" > test

# Read the created file
$ cat test
>hello

>world

# In-between comment

# Read a file that doesn't exist
1$ cat unknown
2>cat: unknown: No such file or directory

$ read input
<Input content

$ ls
>total 8
>-rw-r--r--  1 mickael     6B Dec 28 10:16 test
`, ScriptNode{Nodes: []Node{
		CommentNode{Content: "Script header"},
		&CommandNode{
			Comment: CommentNode{Content: "Create a file"},
			Cmd:     "echo \"hello\\nworld\" > test",
		},
		&CommandNode{
			Comment: CommentNode{Content: "Read the created file"},
			Cmd:     "cat test",
			Stdout:  DataNode{Content: "hello\nworld\n"},
		},
		CommentNode{Content: "In-between comment"},
		&CommandNode{
			Comment:  CommentNode{Content: "Read a file that doesn't exist"},
			Cmd:      "cat unknown",
			ExitCode: 1,
			Stderr:   DataNode{Content: "cat: unknown: No such file or directory\n"},
		},
		&CommandNode{
			Cmd:   "read input",
			Stdin: DataNode{Content: "Input content\n"},
		},
		&CommandNode{
			Cmd:    "ls",
			Stdout: DataNode{Content: "total 8\n-rw-r--r--  1 mickael     6B Dec 28 10:16 test\n"},
		},
	}})
}

func TestParseScriptRequireDataNodeUnderACommand(t *testing.T) {
	testParseScriptErr(t, ">data", "unexpected data line before any command: `data\n`")
}

func testParseScript(t *testing.T, content string, expected ScriptNode) {
	actual, err := ParseScript(content)
	assert.Nil(t, err)
	assert.Equal(t, actual, expected)
}

func testParseScriptErr(t *testing.T, content string, msg string) {
	_, err := ParseScript(content)
	assert.Err(t, err, msg)
}

func TestParseLinesEmpty(t *testing.T) {
	testParseLines(t, "   ", []Line{BlankLine{}})
}

func TestParseLinesBlank(t *testing.T) {
	testParseLines(t, "\n\n", []Line{BlankLine{}})
}

func TestParseLinesComment(t *testing.T) {
	testParseLines(t, "# Comment on one line", []Line{
		CommentLine{Content: "Comment on one line"},
	})
}

func TestParseLinesEmptyComment(t *testing.T) {
	testParseLines(t, "#", []Line{
		CommentLine{Content: ""},
	})
}

func TestParseLinesMultilineComment(t *testing.T) {
	testParseLines(t, "# Comment written \n   #on several\n#   lines ", []Line{
		CommentLine{Content: "Comment written\non several\nlines"},
	})
}

func TestParseLinesInlineComment(t *testing.T) {
	testParseLinesErr(t, " prefix # comment", "a comment must start on its own line, line 1")
}

func TestParseLinesCommand(t *testing.T) {
	testParseLines(t, "$  echo 'hello world' ", []Line{
		CommandLine{Cmd: "echo 'hello world'"},
	})
}

func TestParseLinesEmptyCommand(t *testing.T) {
	testParseLinesErr(t, "$   ", "unexpected empty command, line 1")
}

func TestParseLinesMultipleCommands(t *testing.T) {
	testParseLines(t, `$  echo "hello world"
  $zk list --link-to test.md
`, []Line{
		CommandLine{Cmd: `echo "hello world"`},
		CommandLine{Cmd: `zk list --link-to test.md`},
	})
}

func TestParseLinesCommandWithExitStatus(t *testing.T) {
	testParseLines(t, "1$  cmd1\n255$cmd2", []Line{
		CommandLine{Cmd: "cmd1", ExitCode: 1},
		CommandLine{Cmd: "cmd2", ExitCode: 255},
	})
}

func TestParseLinesCommandInvalidPrefix(t *testing.T) {
	testParseLinesErr(t, " pref$  cmd", "invalid command prefix: `pref`")
}

func TestParseLinesStdin(t *testing.T) {
	testParseLines(t, " <  Input sent to the program ", []Line{
		DataLine{FD: Stdin, Content: "  Input sent to the program \n"},
	})
}

func TestParseLinesEmptyStdin(t *testing.T) {
	testParseLines(t, "	<", []Line{
		DataLine{FD: Stdin, Content: "\n"},
	})
}

func TestParseLinesMultipleStdin(t *testing.T) {
	testParseLines(t, `	<  Input sent to a program 
<	
< which spans
<several lines    

<Another one
`, []Line{
		DataLine{FD: Stdin, Content: "  Input sent to a program \n	\n which spans\nseveral lines    \n"},
		BlankLine{},
		DataLine{FD: Stdin, Content: "Another one\n"},
	})
}

func TestParseLinesStdinInvalidPrefix(t *testing.T) {
	testParseLinesErr(t, " pref<  Error", "invalid data prefix: `pref`")
}

func TestParseLinesStdout(t *testing.T) {
	testParseLines(t, " >  Output from a program ", []Line{
		DataLine{FD: Stdout, Content: "  Output from a program \n"},
	})
}

func TestParseLinesStdoutWithoutNewline(t *testing.T) {
	testParseLines(t, " >  Output from a program \\", []Line{
		DataLine{FD: Stdout, Content: "  Output from a program "},
	})
}

func TestParseLinesEmptyStdout(t *testing.T) {
	testParseLines(t, "	>", []Line{
		DataLine{FD: Stdout, Content: "\n"},
	})
}

func TestParseLinesMultipleStdout(t *testing.T) {
	testParseLines(t, `	>  Output from a program 
>	
> which spans
>several lines    

>Another one
`, []Line{
		DataLine{FD: Stdout, Content: "  Output from a program \n	\n which spans\nseveral lines    \n"},
		BlankLine{},
		DataLine{FD: Stdout, Content: "Another one\n"},
	})
}

func TestParseLinesStdoutInvalidPrefix(t *testing.T) {
	testParseLinesErr(t, " a>  Error", "invalid data prefix: `a`")
}

func TestParseLinesStderr(t *testing.T) {
	testParseLines(t, " 2>  Error from a program ", []Line{
		DataLine{FD: Stderr, Content: "  Error from a program \n"},
	})
}

func TestParseLinesEmptyStderr(t *testing.T) {
	testParseLines(t, "	2>", []Line{
		DataLine{FD: Stderr, Content: "\n"},
	})
}

func TestParseLinesMultipleStderr(t *testing.T) {
	testParseLines(t, `	2>  Error from a program 
2>	
2> which spans
2>several lines    

2>Another one
`, []Line{
		DataLine{FD: Stderr, Content: "  Error from a program \n	\n which spans\nseveral lines    \n"},
		BlankLine{},
		DataLine{FD: Stderr, Content: "Another one\n"},
	})
}

func TestParseLinesCompleteExample(t *testing.T) {
	testParseLines(t, `# Create a file
$ echo "hello" > test

# Read the created file
$ cat test
>hello

# Read a file that doesn't exist
1$ cat unknown
2>cat: unknown: No such file or directory

$ read input
<Input content

$ ls
>total 8
>-rw-r--r--  1 mickael     6B Dec 28 10:16 test
`, []Line{
		CommentLine{Content: "Create a file"},
		CommandLine{Cmd: `echo "hello" > test`},
		BlankLine{},
		CommentLine{Content: "Read the created file"},
		CommandLine{Cmd: "cat test"},
		DataLine{FD: Stdout, Content: "hello\n"},
		BlankLine{},
		CommentLine{Content: "Read a file that doesn't exist"},
		CommandLine{Cmd: "cat unknown", ExitCode: 1},
		DataLine{FD: Stderr, Content: "cat: unknown: No such file or directory\n"},
		BlankLine{},
		CommandLine{Cmd: "read input"},
		DataLine{FD: Stdin, Content: "Input content\n"},
		BlankLine{},
		CommandLine{Cmd: "ls"},
		DataLine{FD: Stdout, Content: "total 8\n-rw-r--r--  1 mickael     6B Dec 28 10:16 test\n"},
	})
}

func testParseLines(t *testing.T, content string, expected []Line) {
	actual, err := parseLines(content)
	assert.Nil(t, err)
	assert.Equal(t, actual, expected)
}

func testParseLinesErr(t *testing.T, content string, msg string) {
	_, err := parseLines(content)
	assert.Err(t, err, msg)
}
