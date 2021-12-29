package tesh

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

func ParseSuite(rootDir string) (TestSuiteNode, error) {
	var suite TestSuiteNode

	err := filepath.Walk(rootDir, func(abs string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		path, err := filepath.Rel(rootDir, abs)
		if err != nil {
			return err
		}
		if filepath.Ext(path) != ".tesh" {
			return nil
		}

		test, err := ParseTestFile(abs)
		if err != nil {
			return err
		}

		test.Name = path
		suite.Tests = append(suite.Tests, test)
		return nil
	})

	return suite, err
}

func ParseTestFile(path string) (TestNode, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return TestNode{}, err
	}
	return ParseTest(string(data))
}

func ParseTest(content string) (TestNode, error) {
	script := TestNode{}
	lines, err := parseLines(content)
	if err != nil {
		return script, err
	}

	comment := CommentNode{}
	var cmd *CommandNode

	flushComment := func() {
		if !comment.IsEmpty() {
			script.Children = append(script.Children, comment)
			comment = CommentNode{}
		}
	}

	for _, line := range lines {
		switch line := line.(type) {
		case BlankLine:
			flushComment()

		case CommandLine:
			cmd = &CommandNode{
				Cmd:      line.Cmd,
				ExitCode: line.ExitCode,
				Comment:  comment,
			}
			script.Children = append(script.Children, cmd)
			comment = CommentNode{}

		case CommentLine:
			flushComment()
			comment.Content = line.Content

		case DataLine:
			// For now we discard any comment above data.
			comment = CommentNode{}
			if cmd == nil {
				return script, fmt.Errorf("unexpected data line before any command: `%s`", line.Content)
			}
			switch line.FD {
			case Stdin:
				cmd.Stdin = cmd.Stdin.Append(line)
			case Stdout:
				cmd.Stdout = cmd.Stdout.Append(line)
			case Stderr:
				cmd.Stderr = cmd.Stderr.Append(line)
			}

		default:
			panic(fmt.Sprintf("unknown Line statement: %+v", line))
		}
	}

	return script, nil
}

func parseLines(content string) ([]Line, error) {
	stmts := []Line{}

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineno := 0
	var prevStmt Line
	for scanner.Scan() {
		lineno += 1
		stmt, err := parseLine(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("%w, line %d", err, lineno)
		}
		if prevStmt == nil {
			prevStmt = stmt
		} else if mergedStmt, ok := prevStmt.Merge(stmt); ok {
			prevStmt = mergedStmt
		} else {
			stmts = append(stmts, prevStmt)
			prevStmt = stmt
		}
	}
	if prevStmt != nil {
		stmts = append(stmts, prevStmt)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stmts, nil
}

func parseLine(line string) (Line, error) {
	if strings.TrimSpace(line) == "" {
		return BlankLine{}, nil
	}

	var prefix string
	for i, char := range line {
		if unicode.IsSpace(char) {
			continue
		}

		switch char {
		case '#':
			return parseComment(prefix, line[i+1:])
		case '$':
			return parseCommand(prefix, line[i+1:])
		case '<':
			return parseInput(prefix, line[i+1:])
		case '>':
			return parseOutput(prefix, line[i+1:])
		default:
			prefix += string(char)
		}
	}

	return nil, fmt.Errorf("unexpected statement: `%s`", line)
}

func parseComment(prefix, line string) (Line, error) {
	if prefix != "" {
		return nil, fmt.Errorf("a comment must start on its own line")
	}
	return CommentLine{Content: strings.TrimSpace(line)}, nil
}

func parseCommand(prefix, line string) (Line, error) {
	exitCode := 0
	if prefix != "" {
		var err error
		exitCode, err = strconv.Atoi(prefix)
		if err != nil {
			return nil, fmt.Errorf("invalid command prefix: `%s`", prefix)
		}
	}
	cmd := strings.TrimSpace(line)
	if cmd == "" {
		return nil, fmt.Errorf("unexpected empty command")
	}
	return CommandLine{
		Cmd:      cmd,
		ExitCode: exitCode,
	}, nil
}

func parseInput(prefix, line string) (Line, error) {
	if prefix != "" {
		return nil, fmt.Errorf("invalid data prefix: `%s`", prefix)
	}
	return parseDataLine(line, Stdin), nil
}

func parseOutput(prefix, line string) (Line, error) {
	fd := Stdout
	if prefix == "2" {
		fd = Stderr
	} else if prefix != "" {
		return nil, fmt.Errorf("invalid data prefix: `%s`", prefix)
	}
	return parseDataLine(line, fd), nil
}

func parseDataLine(line string, fd FD) DataLine {
	if strings.HasSuffix(line, "\\") {
		line = strings.TrimSuffix(line, "\\")
	} else {
		line += "\n"
	}
	return DataLine{FD: fd, Content: line}
}
