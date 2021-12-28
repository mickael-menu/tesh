package main

import (
	"bufio"
	"fmt"
	"strings"
	"unicode"
)

func ParseScript(content string) (ScriptNode, error) {
	script := ScriptNode{}
	lines, err := parseLines(content)
	if err != nil {
		return script, err
	}

	comment := CommentNode{}
	var cmd *CommandNode

	flushComment := func() {
		if !comment.IsEmpty() {
			script.Nodes = append(script.Nodes, comment)
			comment = CommentNode{}
		}
	}

	for _, line := range lines {
		switch line := line.(type) {
		case BlankLine:
			flushComment()

		case CommandLine:
			cmd = &CommandNode{
				Cmd:     line.Cmd,
				Comment: comment,
			}
			script.Nodes = append(script.Nodes, cmd)
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
	if prefix != "" {
		return nil, fmt.Errorf("invalid command prefix: `%s`", prefix)
	}
	cmd := strings.TrimSpace(line)
	if cmd == "" {
		return nil, fmt.Errorf("unexpected empty command")
	}
	return CommandLine{Cmd: cmd}, nil
}

func parseInput(prefix, line string) (Line, error) {
	if prefix != "" {
		return nil, fmt.Errorf("invalid data prefix: `%s`", prefix)
	}
	return DataLine{FD: Stdin, Content: line}, nil
}

func parseOutput(prefix, line string) (Line, error) {
	fd := Stdout
	if prefix == "2" {
		fd = Stderr
	} else if prefix != "" {
		return nil, fmt.Errorf("invalid data prefix: `%s`", prefix)
	}
	return DataLine{FD: fd, Content: line}, nil
}
