package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExec(t *testing.T) {
	t.Run("WithValidCommand", func(t *testing.T) {
		cmd := Cmd{
			Name: "echo",
			Args: []string{"Hello, World!"},
		}
		err := Exec(cmd)
		assert.NoError(t, err)
	})

	t.Run("WithInvalidCommand", func(t *testing.T) {
		cmd := Cmd{
			Name: "invalid_command",
		}
		err := Exec(cmd)
		assert.Error(t, err)
	})

	t.Run("WithRedirection", func(t *testing.T) {
		var stdout bytes.Buffer
		cmd := Cmd{
			Name:   "echo",
			Args:   []string{"Hello, World!"},
			Stdout: &stdout,
		}
		err := Exec(cmd)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, World!\n", stdout.String())
	})

	t.Run("WithDir", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		cmd := Cmd{
			Name: "touch",
			Args: []string{"testfile"},
			Dir:  tmpDir,
		}
		err = Exec(cmd)
		assert.NoError(t, err)

		_, err = os.Stat(tmpDir + "/testfile")
		assert.False(t, os.IsNotExist(err))
	})
}

func TestExecOut(t *testing.T) {
	t.Run("WithValidCommand", func(t *testing.T) {
		cmd := Cmd{
			Name: "echo",
			Args: []string{"Hello, World!"},
		}
		out, err := ExecOut(cmd)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, World!\n", out)
	})

	t.Run("WithInvalidCommand", func(t *testing.T) {
		cmd := Cmd{
			Name: "invalid_command",
		}
		_, err := ExecOut(cmd)
		assert.Error(t, err)
	})
}
