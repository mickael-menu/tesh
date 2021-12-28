package main

type Command struct {
	Comment string
}

type Stmt interface {
	Merge(other Stmt) (Stmt, bool)
}

type BlankStmt struct{}

func (s BlankStmt) Merge(other Stmt) (Stmt, bool) {
	_, ok := other.(BlankStmt)
	return s, ok
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
