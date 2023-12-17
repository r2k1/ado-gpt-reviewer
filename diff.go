package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func GitSync() error {
	if err := os.MkdirAll(cfg.GitRepoPath, 0755); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}
	empty, err := IsDirEmpty(cfg.GitRepoPath)
	if err != nil {
		return err
	}
	if empty {
		repo := fmt.Sprintf("https://%s:%s@%s", cfg.User, cfg.PersonalAccessToken, cfg.GitRepo)
		err := Exec(Cmd{
			Dir:  cfg.GitRepoPath,
			Name: "git",
			Args: []string{"clone", repo, "."},
		})
		if err != nil {
			return err
		}
	}

	return Exec(Cmd{
		Dir:  cfg.GitRepoPath,
		Name: "git",
		Args: []string{"fetch"},
	})
}

func GetDiff(targetBranch, sourceSHA string) (string, error) {
	if err := GitSync(); err != nil {
		return "", err
	}

	targetBranch = "origin/" + targetBranch
	mergeBaseSha, err := ExecOut(Cmd{
		Dir:  cfg.GitRepoPath,
		Name: "git",
		Args: []string{"merge-base", targetBranch, sourceSHA},
	})
	if err != nil {
		return "", err
	}
	mergeBaseSha = strings.TrimSpace(mergeBaseSha)
	return ExecOut(Cmd{
		Dir:  cfg.GitRepoPath,
		Name: "git",
		Args: []string{"diff", mergeBaseSha + ".." + sourceSHA},
	})
}

// IsDirEmpty checks if a directory is empty. Returns true if the directory is empty, false otherwise.
func IsDirEmpty(dirPath string) (bool, error) {
	f, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Read the first entry in the directory
	_, err = f.Readdir(1)

	// If the folder is empty, Readdir returns io.EOF
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
