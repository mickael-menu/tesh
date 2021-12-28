package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Run(script ScriptNode) error {
	for _, node := range script.Nodes {
		switch node := node.(type) {
		case CommentNode:
			fmt.Println(node.Dump())
		case *CommandNode:
			err := runCmd(*node)
			if err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("unknown script Node: %s", node.Dump()))
		}
	}

	return nil
}

func runCmd(node CommandNode) error {
	fmt.Println(node.DumpShort())

	if node.IsEmpty() {
		return fmt.Errorf("unexpected empty command")
	}

	cmd := cmdFromString(node.Cmd)
	if !node.Stdin.IsEmpty() {
		cmd.Stdin = strings.NewReader(node.Stdin.Content)
	}
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
				return fmt.Errorf("command exited with status %d, but expected %d", status, node.ExitCode)
			}
		} else {
			return err
		}
	} else {
		if node.ExitCode != 0 {
			return fmt.Errorf("command exited with status 0, but expected %d", node.ExitCode)
		}
	}

	stdout := string(stdoutBuf.Bytes())
	expectedStdout := node.Stdout.Dump()
	if stdout != expectedStdout {
		return fmt.Errorf("expected stdout `%s`, received `%s`", expectedStdout, stdout)
	}

	stderr := string(stderrBuf.Bytes())
	expectedStderr := node.Stderr.Dump()
	if stderr != expectedStderr {
		return fmt.Errorf("expected stderr `%s`, received `%s`", expectedStderr, stderr)
	}

	return nil
}

func cmdFromString(command string, args ...string) *exec.Cmd {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	args = append([]string{"-c", command, "--"}, args...)
	return exec.Command(shell, args...)
}
