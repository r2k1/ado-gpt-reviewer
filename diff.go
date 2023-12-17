package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func GitClone() {
	// create tmp folder
	_ = os.Mkdir(cfg.GitRepoPath, 0755)
	empty, err := IsDirEmpty(cfg.GitRepoPath)
	checkErr(err)
	if empty {
		// TODO: this operation is slooow it downloads like 2 gigs, maybe there is a way to optimize it?
		repo := fmt.Sprintf("https://%s:%s@%s", cfg.User, cfg.PersonalAccessToken, cfg.GitRepo)
		MustExecOut(Cmd{
			Dir:  cfg.GitRepoPath,
			Name: "git",
			Args: []string{"clone", repo, "."},
		})
	}

	MustExec(Cmd{
		Dir:  cfg.GitRepoPath,
		Name: "git",
		Args: []string{"fetch"},
	})
}

// TODO: add support for non-master branch
func GetDiff(targetBranch, sourceSHA string) string {
	GitClone()
	targetBranch = "origin/" + targetBranch
	mergeBaseSha := MustExecOut(Cmd{
		Dir:  cfg.GitRepoPath,
		Name: "git",
		Args: []string{"merge-base", targetBranch, sourceSHA},
	})
	mergeBaseSha = strings.TrimSpace(mergeBaseSha)
	return MustExecOut(Cmd{
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
