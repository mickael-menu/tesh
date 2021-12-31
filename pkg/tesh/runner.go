package tesh

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aymerick/raymond"
	"github.com/mickael-menu/tesh/pkg/internal/handlebars"
	_ "github.com/mickael-menu/tesh/pkg/internal/handlebars"
	"github.com/mickael-menu/tesh/pkg/internal/util/errors"
	executil "github.com/mickael-menu/tesh/pkg/internal/util/exec"
	"github.com/mickael-menu/tesh/pkg/internal/util/paths"
)

type RunCallbacks struct {
	OnStartTest  func(test TestNode)
	OnUpdateTest func(test TestNode)
	OnFinishTest func(test TestNode, err error)

	OnStartCommand  func(test TestNode, cmd CommandNode, config RunConfig)
	OnFinishCommand func(test TestNode, cmd CommandNode, config RunConfig, err error)

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
	Callbacks  RunCallbacks
	context    map[string]interface{}
}

func (c RunConfig) Context() map[string]interface{} {
	context := c.context
	if context == nil {
		context = map[string]interface{}{}
	}

	context["working-dir"] = c.WorkingDir
	return context
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
	targetDir, err = paths.Canonical(targetDir)
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

loop:
	for _, node := range test.Children {
		switch node := node.(type) {
		case CommentNode:
			if callbacks.OnComment != nil {
				callbacks.OnComment(test, node.Content)
			}
		case *CommandNode:
			if callbacks.OnStartCommand != nil {
				callbacks.OnStartCommand(test, *node, config)
			}
			config.WorkingDir, err = runCmd(node, config, &hasChanges)
			if callbacks.OnFinishCommand != nil {
				callbacks.OnFinishCommand(test, *node, config, err)
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

func runCmd(node *CommandNode, config RunConfig, hasChanges *bool) (string, error) {
	if node.IsEmpty() {
		return config.WorkingDir, fmt.Errorf("unexpected empty command")
	}

	if strings.HasPrefix(node.Cmd, "cd ") {
		path := strings.TrimPrefix(node.Cmd, "cd ")
		path, err := expandString(path, config.Context())
		return filepath.Join(config.WorkingDir, path), err

	} else {
		err := runShellCmd(node, config, hasChanges)
		return config.WorkingDir, err
	}
}

func runShellCmd(sourceNode *CommandNode, config RunConfig, hasChanges *bool) error {
	node, err := expandNode(*sourceNode, config.Context())
	if err != nil {
		return err
	}

	cmd := executil.CommandFromString(node.Cmd)
	cmd.Dir = config.WorkingDir
	if !node.Stdin.IsEmpty() {
		cmd.Stdin = strings.NewReader(node.Stdin.Content)
	}
	if config.WorkingDir != "" {
		cmd.Env = []string{"PATH=" + config.WorkingDir + ":" + os.Getenv("PATH")}
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
	matched, matchErr := matchString(stderr, expectedStderr)
	if matchErr != nil {
		return matchErr
	}
	if !matched {
		if config.Update {
			sourceNode.Stderr.Content = stderr
			*hasChanges = true
		} else {
			_, expected := expandRegexes(expectedStderr)
			return DataAssertError{
				FD:       Stderr,
				Received: stderr,
				Expected: expected,
			}
		}
	}

	stdout := string(stdoutBuf.Bytes())
	stdout = strings.TrimLeft(stdout, "\r")
	expectedStdout := node.Stdout.Dump()
	matched, matchErr = matchString(stdout, expectedStdout)
	if matchErr != nil {
		return matchErr
	}
	if !matched {
		if config.Update {
			sourceNode.Stdout.Content = stdout
			*hasChanges = true
		} else {
			_, expected := expandRegexes(expectedStdout)
			return DataAssertError{
				FD:       Stdout,
				Received: stdout,
				Expected: expected,
			}
		}
	}

	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			status := err.ExitCode()
			if status != node.ExitCode {
				if config.Update {
					sourceNode.ExitCode = status
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
				sourceNode.ExitCode = 0
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

func matchString(actual string, expected string) (bool, error) {
	if actual == "" && expected == "" {
		return true, nil
	}
	hasRegexes, expected := expandRegexes(expected)
	if !hasRegexes {
		return actual == expected, nil
	}

	reg, err := regexp.Compile(expected)
	if err != nil {
		return false, err
	}
	res := reg.Match([]byte(actual))
	return res, nil
}

func expandRegexes(s string) (bool, string) {
	res := regexp.QuoteMeta(s)
	res, hasRegex := handlebars.ExpandRegexes(res)
	if !hasRegex {
		return false, s
	}
	return true, "^" + res + "$"
}
