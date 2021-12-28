package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Run(script ScriptNode) error {
	var wd string
	var err error

	for _, node := range script.Nodes {
		switch node := node.(type) {
		case CommentNode:
			fmt.Println(node.Dump())
		case *CommandNode:
			wd, err = runCmd(*node, wd)
			if err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("unknown script Node: %s", node.Dump()))
		}
	}

	return nil
}

func runCmd(node CommandNode, wd string) (string, error) {
	fmt.Println(node.DumpShort())

	if node.IsEmpty() {
		return wd, fmt.Errorf("unexpected empty command")
	}

	cmd := cmdFromString(node.Cmd)
	cmd.Dir = wd
	if !node.Stdin.IsEmpty() {
		cmd.Stdin = strings.NewReader(node.Stdin.Content)
	}
	cmd.Env = []string{"PATH=/bin:" + os.Getenv("PATH")}
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			status := err.ExitCode()
			if status != node.ExitCode {
				fmt.Println(string(err.Stderr))
				return wd, fmt.Errorf("command exited with status %d, but expected %d", status, node.ExitCode)
			}
		} else {
			return wd, err
		}
	} else {
		if node.ExitCode != 0 {
			return wd, fmt.Errorf("command exited with status 0, but expected %d", node.ExitCode)
		}
	}

	stdout := string(stdoutBuf.Bytes())
	stdout = strings.TrimRight(stdout, "\n")
	wd = stdout[strings.LastIndex(stdout, "\n")+1:]
	stdout = strings.TrimSuffix(stdout, wd)
	expectedStdout := node.Stdout.Dump()
	if stdout != expectedStdout {
		return wd, fmt.Errorf("expected stdout `%s`, received `%s`", expectedStdout, stdout)
	}

	stderr := string(stderrBuf.Bytes())
	expectedStderr := node.Stderr.Dump()
	if stderr != expectedStderr {
		return wd, fmt.Errorf("expected stderr `%s`, received `%s`", expectedStderr, stderr)
	}

	return wd, nil
}

func cmdFromString(command string, args ...string) *exec.Cmd {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	args = append([]string{"-c", command + " && pwd", "--"}, args...)
	return exec.Command(shell, args...)
}
