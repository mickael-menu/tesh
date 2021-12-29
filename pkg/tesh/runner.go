package tesh

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RunCallbacks struct {
	OnStartTest  func(test TestNode)
	OnFinishTest func(test TestNode, err error)

	OnStartCommand  func(test TestNode, cmd CommandNode, wd string)
	OnFinishCommand func(test TestNode, cmd CommandNode, wd string, err error)

	OnComment func(test TestNode, comment string)
}

type ExitCodeAssertError struct {
	Received int
	Expected int
	Stderr   string
}

func (e ExitCodeAssertError) Error() string {
	out := fmt.Sprintf("expected exit code %d, got %d", e.Expected, e.Received)
	if e.Stderr != "" {
		out += ": stderr: " + e.Stderr
	}
	return out
}

type DataAssertError struct {
	FD       FD
	Received string
	Expected string
}

func (e DataAssertError) Error() string {
	return fmt.Sprintf("expected on %s: `%s` got: `%s`", e.FD.String(), e.Expected, e.Received)
}

type RunConfig struct {
	WorkingDir string
	Callbacks  RunCallbacks
}

type RunReport struct {
	FailedCount int
	TotalCount  int
}

func RunSuite(suite TestSuiteNode, config RunConfig) (RunReport, error) {
	report := RunReport{
		TotalCount: len(suite.Tests),
	}

	for _, test := range suite.Tests {
		err := RunTest(test, config)
		if err != nil {
			report.FailedCount += 1
		}
	}

	return report, nil
}

func RunTest(test TestNode, config RunConfig) error {
	callbacks := config.Callbacks

	if callbacks.OnStartTest != nil {
		callbacks.OnStartTest(test)
	}

	var wd string
	var err error

loop:
	for _, node := range test.Children {
		switch node := node.(type) {
		case CommentNode:
			if callbacks.OnComment != nil {
				callbacks.OnComment(test, node.Content)
			}
		case *CommandNode:
			if callbacks.OnStartCommand != nil {
				callbacks.OnStartCommand(test, *node, wd)
			}
			wd, err = runCmd(*node, wd)
			if callbacks.OnFinishCommand != nil {
				callbacks.OnFinishCommand(test, *node, wd, err)
			}
			if err != nil {
				break loop
			}
		default:
			panic(fmt.Sprintf("unknown script Node: %s", node.Dump()))
		}
	}
	if callbacks.OnFinishTest != nil {
		callbacks.OnFinishTest(test, err)
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

	stdout := string(stdoutBuf.Bytes())
	// Sometimes some garbage \r is prepended to stdout/stderr.
	stdout = strings.TrimLeft(stdout, "\r")
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
	stderr = strings.TrimLeft(stderr, "\r")
	expectedStderr := node.Stderr.Dump()
	if stderr != expectedStderr {
		return wd, DataAssertError{
			FD:       Stderr,
			Received: stderr,
			Expected: expectedStderr,
		}
	}

	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			status := err.ExitCode()
			if status != node.ExitCode {
				return wd, ExitCodeAssertError{
					Received: status,
					Expected: node.ExitCode,
					Stderr:   string(err.Stderr),
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
