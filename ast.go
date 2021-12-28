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

func (n CommandNode) DumpShort() string {
	if n.IsEmpty() {
		return ""
	}
	return fmt.Sprintf("%s$ %s\n", n.Comment.Dump(), n.Cmd)
}

func (n CommandNode) Dump() string {
	if n.IsEmpty() {
		return ""
	}

	out := n.DumpShort()
	if !n.Stdin.IsEmpty() {
		out += "< " + n.Stdin.Dump()
	}
	if !n.Stdout.IsEmpty() {
		out += "< " + n.Stdout.Dump()
	}
	if !n.Stderr.IsEmpty() {
		out += "< " + n.Stderr.Dump()
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
	if n.IsEmpty() {
		return ""
	} else {
		return n.Content + "\n"
	}
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
