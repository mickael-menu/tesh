# tesh

`tesh` is a CLI tool and Go library used to test the output of shell commands.

:warning: This is a very crude early proof of concept, use with caution.

A real life `tesh` test suite is used in the [`zk` repository](https://github.com/mickael-menu/zk/tree/main/tests).

## Build

You need a working [Go installation](https://golang.org/) to build `tesh`.

```sh
$ git clone https://github.com/mickael-menu/tesh.git
$ cd tesh
$ go install cmd/tesh.go
```

## Usage

```sh
$ tesh <tests-dir> <working-dir>
```

Run all the `.tesh` files found in the `tests-dir` directory, recursively. The tests are run from a copy of the given `working-dir`, which can contain test fixtures.

```sh
$ tesh -u <tests-dir> <working-dir>
```

Update the `.tesh` files in place (`stdout` and `stderr` outputs) when encountering a failed test.

```sh
$ tesh -b <tests-dir> <working-dir>
```

Print raw bytes for the expected outputs, in case of failure. Useful for debugging whitespaces.

## Syntax

A `.tesh` file represents a single `tesh` test case, but can contain several commands. Here's a complete example of a `.tesh` file:

```sh
# Test output on stdout
$ echo "hello\nworld"
>hello
>world

# Test output on stderr
1$ cat not-found
2>cat: not-found: No such file or directory

# Test input from stdin
$ cat -n
<Testing input
<on several lines
>     1	Testing input
>     2	on several lines

# Test exit code
42$ exit 42
```

### Comments

Only single-line comments are supported, with `#`. End-of-line comments are not possible.

### Commands

Each command must start with `$`, followed by a shell statement. You can use pipes and shell variables, as the statement will be passed to `$SHELL -c`.

An exit code of `0` is expected, unless you prefix the `$` with a failure code, e.g. `1$ cat not-found`.

#### `cd` command

The `cd` command is special with `tesh`, it needs to be on its own line and can't be combined with other commands (e.g. with `&&` or `|`).


### Input streams (`stdin`)

You can provide input for a command by prefixing it with `<`. Whitespaces after `<` are significant, including the final newline.

### Output streams (`stdout` on `stderr`)

Use `>` for the expected output on `stdout`, or `2>` for the expected output on `stderr`. Whitespaces after `>` are significant, including the final newline.
If the command doesn't output a final newline, you can use a trailing `\` to match the output.

### Templates

Commands and streams can contain [Handlebars statements](https://handlebarsjs.com/). Some additional helpers are available

#### `match` helper (Regexes)

The `match` helper is useful to check for dynamic content, as you can use a regular expression to verify the content of an output stream.

```
$ uname -rs
>Darwin {{match '[0-9\.]+'}}
```

#### `sh` helper

The `sh` helper can be used to execute a shell command and expand its output in the template.

```
{{sh "echo 'Hello, world!'"}} -> Hello, world!
{{#sh "tr '[a-z]' '[A-Z]'"}}Hello, world!{{/sh}} -> HELLO, WORLD!
```

### Character escaping

Some characters are significant in the commands and streams. If you want to use them literally, you must escape them with `\`:

* `$`, for example when used with shell environment variables.
* `{{`, as it is reserved for Handlebars templates.

