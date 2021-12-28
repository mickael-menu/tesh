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
1$ cat unknown
2>cat: unknown: No such file or directory

$ cat -n
<Input content
>     1	Input content
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
