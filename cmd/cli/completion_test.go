package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionCmd_Run(t *testing.T) {
	t.Run("outputs zsh completions", func(t *testing.T) {
		var out bytes.Buffer
		cmd := CompletionCmd{Shell: "zsh"}

		require.NoError(t, cmd.Run(&Globals{Out: &out}))
		assert.Equal(t, zshCompletion, out.Bytes())
	})

	t.Run("returns output errors", func(t *testing.T) {
		reader, writer := io.Pipe()
		require.NoError(t, reader.Close())
		defer writer.Close()
		cmd := CompletionCmd{Shell: "zsh"}

		require.ErrorIs(t, cmd.Run(&Globals{Out: writer}), io.ErrClosedPipe)
	})

	t.Run("rejects unsupported shells without output", func(t *testing.T) {
		var out bytes.Buffer
		cmd := CompletionCmd{Shell: "bash"}

		require.EqualError(t, cmd.Run(&Globals{Out: &out}), "unsupported shell: bash")
		assert.Empty(t, out.String())
	})
}
