package main

import (
	_ "embed"
	"fmt"
	"pj/internal/util"
)

//go:embed completions/pj.zsh
var zshCompletion []byte

type CompletionCmd struct {
	Shell string `arg:"" enum:"zsh" help:"Shell type (zsh)"`
}

func (cmd *CompletionCmd) Run(g *Globals) error {
	switch cmd.Shell {
	case "zsh":
		assert.Success(g.Out.Write(zshCompletion))
	default:
		return fmt.Errorf("unsupported shell: %s", cmd.Shell)
	}

	return nil
}
