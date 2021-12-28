package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RunCallbacks struct {
	OnStartScript  func(script ScriptNode)
	OnFinishScript func(script ScriptNode, err error)

	OnStartCommand  func(script ScriptNode, cmd CommandNode, wd string)
	OnFinishCommand func(script ScriptNode, cmd CommandNode, wd string, err error)

	OnComment func(script ScriptNode, comment string)
}

type ExitCodeAssertError struct {
	Received int
	Expected int
}

func (e ExitCodeAssertError) Error() string {
	return fmt.Sprintf("expected exit code %d, got %d", e.Expected, e.Received)
}

type DataAssertError struct {
	FD       FD
	Received string
	Expected string
}

func (e DataAssertError) Error() string {
	return fmt.Sprintf("expected on %s: %s got: %s", e.FD.String(), e.Expected, e.Received)
}

func Run(script ScriptNode, callbacks RunCallbacks) error {
	if callbacks.OnStartScript != nil {
		callbacks.OnStartScript(script)
	}

	var wd string
	var err error

loop:
	for _, node := range script.Nodes {
		switch node := node.(type) {
		case CommentNode:
			if callbacks.OnComment != nil {
				callbacks.OnComment(script, node.Content)
			}
		case *CommandNode:
			if callbacks.OnStartCommand != nil {
				callbacks.OnStartCommand(script, *node, wd)
			}
			wd, err = runCmd(*node, wd)
			if callbacks.OnFinishCommand != nil {
				callbacks.OnFinishCommand(script, *node, wd, err)
			}
			if err != nil {
				break loop
			}
		default:
			panic(fmt.Sprintf("unknown script Node: %s", node.Dump()))
		}
	}
	if callbacks.OnFinishScript != nil {
		callbacks.OnFinishScript(script, err)
	}

	return err
}

func runCmd(node CommandNode, wd string) (string, error) {
	if node.IsEmpty() {
		return wd, fmt.Errorf("unexpected empty command")
	}

	cmd := cmdFromString(node.Cmd)
	cmd.Dir = wd
	if !node.Stdin.IsEmpty() {
		cmd.Stdin = strings.NewReader(node.Stdin.Content)
	}
	// cmd.Env = []string{"PATH=/bin:" + os.Getenv("PATH")}
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
				return wd, ExitCodeAssertError{
					Received: status,
					Expected: node.ExitCode,
				}
			}
		} else {
			return wd, err
		}
	} else {
		if node.ExitCode != 0 {
			return wd, ExitCodeAssertError{
				Received: 0,
				Expected: node.ExitCode,
			}
		}
	}

	stdout := string(stdoutBuf.Bytes())
	stdout = strings.TrimRight(stdout, "\n")
	wd = stdout[strings.LastIndex(stdout, "\n")+1:]
	stdout = strings.TrimSuffix(stdout, wd)
	expectedStdout := node.Stdout.Dump()
	if stdout != expectedStdout {
		return wd, DataAssertError{
			FD:       Stdout,
			Received: stdout,
			Expected: expectedStdout,
		}
	}

	stderr := string(stderrBuf.Bytes())
	expectedStderr := node.Stderr.Dump()
	if stderr != expectedStderr {
		return wd, DataAssertError{
			FD:       Stderr,
			Received: stderr,
			Expected: expectedStderr,
		}
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
