package main

import (
	"fmt"
	"os"
)

func main() {
	// var test Stmt
	// test = CommentStmt{Text: "hello"}
	// fmt.Printf("Hello, world: %+v\n", test)
	content := `# Script header

# Create a file
$ echo "hello\nworld" > test

# Read the created file
$ cat test
>hello

>world

# In-between comment

# Read a file that doesn't exist
$ cat unknown
2>cat: unknown: No such file or directory

$ read input
<Input content

$ ls
>total 8
>-rw-r--r--  1 mickael     6B Dec 28 10:16 test
`
	script, err := ParseScript(content)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}
	err = Run(script)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}
}
