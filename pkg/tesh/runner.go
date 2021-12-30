package tesh

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aymerick/raymond"
	"github.com/mickael-menu/tesh/pkg/internal/util/errors"
)

type RunCallbacks struct {
	OnStartTest  func(test TestNode)
	OnUpdateTest func(test TestNode)
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
	// When true, will overwrite the test to make them pass.
	Update     bool
	WorkingDir string
	Context    map[string]interface{}
	Callbacks  RunCallbacks
}

type RunReport struct {
	FailedCount  int
	UpdatedCount int
	TotalCount   int
}

func RunSuite(suite TestSuiteNode, config RunConfig) (RunReport, error) {
	report := RunReport{
		TotalCount: len(suite.Tests),
	}

	for _, test := range suite.Tests {
		wd, err := setupTempWorkingDir(test.Name, config.WorkingDir)
		if err != nil {
			return report, err
		}
		defer os.RemoveAll(wd)

		testConfig := config
		testConfig.WorkingDir = wd

		testConfig.Callbacks.OnFinishTest = func(test TestNode, err error) {
			if config.Callbacks.OnFinishTest != nil {
				config.Callbacks.OnFinishTest(test, err)
			}
			if err != nil {
				report.FailedCount += 1
			}
		}

		testConfig.Callbacks.OnUpdateTest = func(test TestNode) {
			if config.Callbacks.OnUpdateTest != nil {
				config.Callbacks.OnUpdateTest(test)
			}
			report.UpdatedCount += 1
		}

		_ = RunTest(test, testConfig)
	}

	return report, nil
}

func setupTempWorkingDir(name string, sourceDir string) (string, error) {
	targetDir, err := ioutil.TempDir("", strings.ReplaceAll(name, "/", "-")+"-*")
	if err != nil {
		return "", err
	}
	if sourceDir == "" {
		return targetDir, nil
	}

	err = filepath.Walk(sourceDir, func(sourcePath string, info os.FileInfo, err error) error {
		if sourcePath == sourceDir {
			return nil
		}

		wrap := errors.Wrapperf("walk %s", sourcePath)
		if err != nil {
			return wrap(err)
		}
		sourceName, err := filepath.Rel(sourceDir, sourcePath)
		if err != nil {
			return wrap(err)
		}
		targetPath := filepath.Join(targetDir, sourceName)

		if info.IsDir() {
			return wrap(os.Mkdir(targetPath, os.ModePerm))
		} else {
			sourceData, err := ioutil.ReadFile(sourcePath)
			if err != nil {
				return wrap(err)
			}
			err = ioutil.WriteFile(targetPath, sourceData, os.ModePerm)
			if err != nil {
				return wrap(err)
			}
		}
		return nil
	})

	return targetDir, err
}

func RunTest(test TestNode, config RunConfig) error {
	callbacks := config.Callbacks

	if callbacks.OnStartTest != nil {
		callbacks.OnStartTest(test)
	}

	var err error
	hasChanges := false
	wd := config.WorkingDir

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
			wd, err = runCmd(node, wd, config, &hasChanges)
			if callbacks.OnFinishCommand != nil {
				callbacks.OnFinishCommand(test, *node, wd, err)
			}
			if err != nil {
				break loop
			}
		default:
			panic(fmt.Sprintf("unknown test Node: %s", node.Dump()))
		}
	}
	if callbacks.OnFinishTest != nil {
		callbacks.OnFinishTest(test, err)
	}

	if hasChanges && config.Update {
		err = test.Write()
		if err != nil {
			return err
		}
		if callbacks.OnUpdateTest != nil {
			callbacks.OnUpdateTest(test)
		}
	}

	return err
}

func runCmd(node *CommandNode, wd string, config RunConfig, hasChanges *bool) (string, error) {
	if node.IsEmpty() {
		return wd, fmt.Errorf("unexpected empty command")
	}

	if strings.HasPrefix(node.Cmd, "cd ") {
		path := strings.TrimPrefix(node.Cmd, "cd ")
		path, err := expandString(path, config.Context)
		return filepath.Join(wd, path), err

	} else {
		err := runShellCmd(node, wd, config, hasChanges)
		return wd, err
	}
}

func runShellCmd(sourceNode *CommandNode, wd string, config RunConfig, hasChanges *bool) error {
	node, err := expandNode(*sourceNode, config.Context)
	if err != nil {
		return err
	}

	cmd := cmdFromString(node.Cmd)
	cmd.Dir = wd
	if !node.Stdin.IsEmpty() {
		cmd.Stdin = strings.NewReader(node.Stdin.Content)
	}
	if wd != "" {
		cmd.Env = []string{"PATH=" + wd + ":" + os.Getenv("PATH")}
	}
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	err = cmd.Run()

	stderr := string(stderrBuf.Bytes())
	// Sometimes some garbage \r is prepended to stdout/stderr.
	stderr = strings.TrimLeft(stderr, "\r")
	expectedStderr := node.Stderr.Dump()
	if stderr != expectedStderr {
		if config.Update {
			node.Stderr.Content = stderr
			*hasChanges = true
		} else {
			return DataAssertError{
				FD:       Stderr,
				Received: stderr,
				Expected: expectedStderr,
			}
		}
	}

	stdout := string(stdoutBuf.Bytes())
	stdout = strings.TrimLeft(stdout, "\r")
	expectedStdout := node.Stdout.Dump()
	if stdout != expectedStdout {
		if config.Update {
			node.Stdout.Content = stdout
			*hasChanges = true
		} else {
			return DataAssertError{
				FD:       Stdout,
				Received: stdout,
				Expected: expectedStdout,
			}
		}
	}

	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			status := err.ExitCode()
			if status != node.ExitCode {
				if config.Update {
					node.ExitCode = status
					*hasChanges = true
				} else {
					return ExitCodeAssertError{
						Received: status,
						Expected: node.ExitCode,
						Stderr:   string(err.Stderr),
					}
				}
			}
		} else {
			return err
		}
	} else {
		if node.ExitCode != 0 {
			if config.Update {
				node.ExitCode = 0
				*hasChanges = true
			} else {
				return ExitCodeAssertError{
					Received: 0,
					Expected: node.ExitCode,
				}
			}
		}
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

func expandNode(node CommandNode, context map[string]interface{}) (CommandNode, error) {
	var err error
	node.Cmd, err = expandString(node.Cmd, context)
	if err != nil {
		return node, err
	}
	node.Stdin.Content, err = expandString(node.Stdin.Content, context)
	if err != nil {
		return node, err
	}
	node.Stdout.Content, err = expandString(node.Stdout.Content, context)
	if err != nil {
		return node, err
	}
	node.Stderr.Content, err = expandString(node.Stderr.Content, context)
	if err != nil {
		return node, err
	}
	return node, err
}

func expandString(s string, context map[string]interface{}) (string, error) {
	tpl, err := raymond.Parse(s)
	if err != nil {
		return "", err
	}
	return tpl.Exec(context)
}
