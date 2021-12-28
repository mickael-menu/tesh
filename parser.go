package main

import (
	"bufio"
	"fmt"
	"strings"
	"unicode"
)

type Stmt interface {
	Merge(other Stmt) (Stmt, bool)
}

type CommentStmt struct {
	Content string
}

func (s CommentStmt) Merge(other Stmt) (Stmt, bool) {
	if other, ok := other.(CommentStmt); ok {
		return CommentStmt{
			Content: s.Content + "\n" + other.Content,
		}, true
	} else {
		return s, false
	}
}

type CommandStmt struct {
	Cmd string
}

func (s CommandStmt) Merge(other Stmt) (Stmt, bool) {
	return s, false
}

type FD int

const (
	Stdin  FD = 0
	Stdout FD = 1
	Stderr FD = 2
)

type DataStmt struct {
	FD      FD
	Content string
}

func (s DataStmt) Merge(other Stmt) (Stmt, bool) {
	if other, ok := other.(DataStmt); ok && s.FD == other.FD {
		return DataStmt{
			FD:      s.FD,
			Content: s.Content + "\n" + other.Content,
		}, true
	} else {
		return s, false
	}
}

func Parse(content string) ([]Stmt, error) {
	stmts := []Stmt{}

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineno := 0
	var prevStmt Stmt
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

func parseLine(line string) (Stmt, error) {
	if strings.TrimSpace(line) == "" {
		return nil, nil
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

func parseComment(prefix, line string) (Stmt, error) {
	if prefix != "" {
		return nil, fmt.Errorf("a comment must start on its own line")
	}
	return CommentStmt{Content: strings.TrimSpace(line)}, nil
}

func parseCommand(prefix, line string) (Stmt, error) {
	if prefix != "" {
		return nil, fmt.Errorf("invalid command prefix: `%s`", prefix)
	}
	cmd := strings.TrimSpace(line)
	if cmd == "" {
		return nil, fmt.Errorf("unexpected empty command")
	}
	return CommandStmt{Cmd: cmd}, nil
}

func parseInput(prefix, line string) (Stmt, error) {
	if prefix != "" {
		return nil, fmt.Errorf("invalid data prefix: `%s`", prefix)
	}
	return DataStmt{FD: Stdin, Content: line}, nil
}

func parseOutput(prefix, line string) (Stmt, error) {
	fd := Stdout
	if prefix == "2" {
		fd = Stderr
	} else if prefix != "" {
		return nil, fmt.Errorf("invalid data prefix: `%s`", prefix)
	}
	return DataStmt{FD: fd, Content: line}, nil
}
