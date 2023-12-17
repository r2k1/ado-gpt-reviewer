package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Git struct {
	RepoURL string
	Dir     string
}

func (g *Git) Sync() error {
	if err := os.MkdirAll(g.Dir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}
	empty, err := isDirEmpty(g.Dir)
	if err != nil {
		return err
	}
	if empty {
		err := Exec(Cmd{
			Dir:     g.Dir,
			Name:    "git",
			NoPrint: true, // gitURL contains token, don't print it
			Args:    []string{"clone", g.RepoURL, "."},
		})
		if err != nil {
			return err
		}
	}

	return Exec(Cmd{
		Dir:  g.Dir,
		Name: "git",
		Args: []string{"fetch"},
	})
}

// Diff returns a diff similar to a pull request
func (g *Git) Diff(targetBranch, sourceSHA string) (string, error) {
	targetBranch = "origin/" + targetBranch
	mergeBaseSha, err := ExecOut(Cmd{
		Dir:  g.Dir,
		Name: "git",
		Args: []string{"merge-base", targetBranch, sourceSHA},
	})
	if err != nil {
		return "", err
	}
	mergeBaseSha = strings.TrimSpace(mergeBaseSha)
	return ExecOut(Cmd{
		Dir:  g.Dir,
		Name: "git",
		Args: []string{"diff", mergeBaseSha + ".." + sourceSHA},
	})
}

// isDirEmpty checks if a directory is empty. Returns true if the directory is empty, false otherwise.
func isDirEmpty(dirPath string) (bool, error) {
	f, err := os.Open(dirPath)
	if err != nil {
		return false, fmt.Errorf("error opening directory: %w", err)
	}
	defer f.Close()

	// Read the first entry in the directory
	_, err = f.Readdir(1)
	// If the folder is empty, Readdir returns io.EOF
	if errors.Is(err, io.EOF) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("error reading directory: %w", err)
	}
	return false, nil
}
