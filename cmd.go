package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Cmd struct {
	Dir     string
	Name    string
	Args    []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	NoPrint bool
}

func Exec(opts Cmd) error {
	// nolint: gosec
	c := exec.Command(opts.Name, opts.Args...)
	c.Dir = opts.Dir

	if opts.Stdout == nil {
		c.Stdout = os.Stdout
	} else {
		c.Stdout = opts.Stdout
	}
	if opts.Stderr == nil {
		c.Stderr = os.Stderr
	} else {
		c.Stderr = opts.Stderr
	}
	if opts.Stdin != nil {
		c.Stdin = opts.Stdin
	}

	colorReset := "\033[0m"
	colorYellow := "\033[33m"
	colorCyan := "\033[36m"

	if !opts.NoPrint {
		fmt.Printf("⚙️ %s%s%s %s%s%s\n", colorCyan, opts.Dir, colorReset, colorYellow, c.String(), colorReset)
	} else {
		fmt.Printf("⚙️ %s[REDACTED COMMAND]%s\n", colorCyan, opts.Dir, colorReset)
	}
	err := c.Run()
	if err != nil {
		return fmt.Errorf("failed to run command %s: %w", c.String(), err)
	}
	return nil
}

// Same as MustExec but returns stdout content
func MustExecOut(opts Cmd) string {
	var out bytes.Buffer
	opts.Stdout = &out
	err := Exec(opts)
	if err != nil {
		panic(err)
	}
	return out.String()
}

func MustExec(opts Cmd) {
	err := Exec(opts)
	if err != nil {
		panic(err)
	}
}
