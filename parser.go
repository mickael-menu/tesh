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

type DataStmt struct {
	// File descriptor
	// 0 = stdin
	// 1 = stdout
	// 2 = stderr
	FD      int
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

	for i, char := range line {
		if unicode.IsSpace(char) {
			continue
		}

		switch char {
		case '#':
			return CommentStmt{Content: strings.TrimSpace(line[i+1:])}, nil
		case '$':
			return parseCommand(line[i+1:])
		case '>':
			return DataStmt{FD: 1, Content: line[i+1:]}, nil
		}
	}

	return nil, fmt.Errorf("unexpected statement: `%s`", line)
}

func parseCommand(line string) (Stmt, error) {
	cmd := strings.TrimSpace(line)
	if cmd == "" {
		return nil, fmt.Errorf("unexpected empty command")
	}
	return CommandStmt{Cmd: cmd}, nil
}
