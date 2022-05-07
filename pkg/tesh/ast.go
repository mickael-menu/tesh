package tesh

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Node interface {
	IsEmpty() bool
	Dump() string
}

type TestSuiteNode struct {
	Tests []TestNode
}

func (n TestSuiteNode) IsEmpty() bool {
	return len(n.Tests) == 0
}

func (n TestSuiteNode) Dump() string {
	out := ""
	for _, test := range n.Tests {
		out += test.Name + ":\n" + test.Dump()
	}
	return out
}

type TestNode struct {
	Name     string
	Path     string
	Children []Node
}

func (n TestNode) IsEmpty() bool {
	return len(n.Children) == 0
}

func (n TestNode) Dump() string {
	out := ""
	for _, node := range n.Children {
		switch node := node.(type) {
		case CommentNode:
			out += node.Dump()
		case *CommandNode:
			out += node.Dump()
		case SpacerNode:
			out += node.Dump()

		default:
			panic(fmt.Sprintf("unknown test Node: %s", node.Dump()))
		}
	}

	return out
}

func (n TestNode) Write() error {
	if n.Path == "" {
		return fmt.Errorf("writing a test requires a path")
	}

	return ioutil.WriteFile(n.Path, []byte(n.Dump()), os.ModePerm)
}

func prefixLines(content string, prefix string) string {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	return prefix + strings.Join(lines, "\n"+prefix)
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
	return prefixLines(n.Content, "# ") + "\n"
}

type CommandNode struct {
	Comment  CommentNode
	Cmd      string
	ExitCode int
	Stdin    DataNode
	Stdout   DataNode
	Stderr   DataNode
}

func (n CommandNode) IsEmpty() bool {
	return n.Cmd == ""
}

func (n CommandNode) Dump() string {
	if n.IsEmpty() {
		return ""
	}

	out := ""
	if !n.Comment.IsEmpty() {
		out += n.Comment.Dump()
	}

	if n.ExitCode != 0 {
		out += fmt.Sprint(n.ExitCode)
	}
	out += "$ " + n.Cmd + "\n"
	if !n.Stdin.IsEmpty() {
		out += prefixLines(n.Stdin.Content, "<") + "\n"
	}
	if !n.Stdout.IsEmpty() {
		out += prefixLines(n.Stdout.Content, ">") + "\n"
	}
	if !n.Stderr.IsEmpty() {
		out += prefixLines(n.Stderr.Content, "2>") + "\n"
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
	return n.Content
}

func (n DataNode) Append(line DataLine) DataNode {
	return DataNode{
		Content: n.Content + line.Content,
	}
}

type SpacerNode struct {
	Lines int
}

func (n SpacerNode) IsEmpty() bool {
	return n.Lines == 0
}

func (n SpacerNode) Dump() string {
	return strings.Repeat("\n", n.Lines)
}

type Line interface {
	Merge(other Line) (Line, bool)
}

type BlankLine struct {
	Count int
}

func (s BlankLine) Merge(other Line) (Line, bool) {
	if o, ok := other.(BlankLine); ok {
		return BlankLine{Count: s.Count + o.Count}, true
	} else {
		return s, false
	}
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
	Cmd      string
	ExitCode int
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

func (fd FD) String() string {
	switch fd {
	case Stdin:
		return "stdin"
	case Stdout:
		return "stdout"
	case Stderr:
		return "stderr"
	default:
		return fmt.Sprintf("%d", fd)
	}
}

type DataLine struct {
	FD      FD
	Content string
}

func (s DataLine) Merge(other Line) (Line, bool) {
	if other, ok := other.(DataLine); ok && s.FD == other.FD {
		return DataLine{
			FD:      s.FD,
			Content: s.Content + other.Content,
		}, true
	} else {
		return s, false
	}
}
