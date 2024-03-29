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
	Stdout  io.Writer
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

	colorReset := "\033[0m"
	colorYellow := "\033[33m"
	colorCyan := "\033[36m"

	if opts.NoPrint {
		fmt.Printf("⚙️ %s%s[REDACTED COMMAND]%s\n", colorCyan, opts.Dir, colorReset)
	} else {
		fmt.Printf("⚙️ %s%s%s %s%s%s\n", colorCyan, opts.Dir, colorReset, colorYellow, c.String(), colorReset)
	}
	err := c.Run()
	if err != nil {
		return fmt.Errorf("failed to run command %s: %w", c.String(), err)
	}
	return nil
}

// ExecOut is same as Exec but returns stdout content
func ExecOut(opts Cmd) (string, error) {
	var out bytes.Buffer
	opts.Stdout = &out
	err := Exec(opts)
	return out.String(), err
}
