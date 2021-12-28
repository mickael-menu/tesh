package main

import "fmt"

type Node interface {
	IsEmpty() bool
	Dump() string
}

type ScriptNode struct {
	Nodes []Node
}

func (n ScriptNode) IsEmpty() bool {
	return len(n.Nodes) == 0
}

func (n ScriptNode) Dump() string {
	out := ""
	for _, node := range n.Nodes {
		out += node.Dump()
	}
	return out
}

type CommentNode struct {
	Content string
}

func (n CommentNode) IsEmpty() bool {
	return n.Content == ""
}

func (n CommentNode) Dump() string {
	if n.IsEmpty() {
		return ""
	}
	return "# " + n.Content + "\n"
}

type CommandNode struct {
	Comment CommentNode
	Cmd     string
	Stdin   DataNode
	Stdout  DataNode
	Stderr  DataNode
}

func (n CommandNode) IsEmpty() bool {
	return n.Cmd == ""
}

func (n CommandNode) Dump() string {
	if n.IsEmpty() {
		return ""
	}
	out := fmt.Sprintf("%s$ %s\n", n.Comment.Dump(), n.Cmd)
	stdin := n.Stdin.Dump()
	if stdin != "" {
		out += "< " + stdin
	}
	stdout := n.Stdout.Dump()
	if stdout != "" {
		out += "< " + stdout
	}
	stderr := n.Stderr.Dump()
	if stderr != "" {
		out += "2> " + stderr
	}
	return out
}

type DataNode struct {
	Content string
}

func (n DataNode) IsEmpty() bool {
	return n.Content == ""
}

func (n DataNode) Dump() string {
	return n.Content + "\n"
}

func (n DataNode) Append(line DataLine) DataNode {
	content := n.Content
	if content != "" {
		content += "\n"
	}
	return DataNode{
		Content: content + line.Content,
	}
}

type Line interface {
	Merge(other Line) (Line, bool)
}

type BlankLine struct{}

func (s BlankLine) Merge(other Line) (Line, bool) {
	_, ok := other.(BlankLine)
	return s, ok
}

type CommentLine struct {
	Content string
}

func (s CommentLine) Merge(other Line) (Line, bool) {
	if other, ok := other.(CommentLine); ok {
		return CommentLine{
			Content: s.Content + "\n" + other.Content,
		}, true
	} else {
		return s, false
	}
}

type CommandLine struct {
	Cmd string
}

func (s CommandLine) Merge(other Line) (Line, bool) {
	return s, false
}

type FD int

const (
	Stdin  FD = 0
	Stdout FD = 1
	Stderr FD = 2
)

type DataLine struct {
	FD      FD
	Content string
}

func (s DataLine) Merge(other Line) (Line, bool) {
	if other, ok := other.(DataLine); ok && s.FD == other.FD {
		return DataLine{
			FD:      s.FD,
			Content: s.Content + "\n" + other.Content,
		}, true
	} else {
		return s, false
	}
}
