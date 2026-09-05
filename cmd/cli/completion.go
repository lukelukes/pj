package main

import (
	_ "embed"
	"fmt"
)

//go:embed completions/pj.zsh
var zshCompletion []byte

type CompletionCmd struct {
	Shell string `arg:"" enum:"zsh" help:"Shell type (zsh)"`
}

func (cmd *CompletionCmd) Run(g *Globals) error {
	switch cmd.Shell {
	case "zsh":
		_, err := g.Out.Write(zshCompletion)
		return err
	default:
		return fmt.Errorf("unsupported shell: %s", cmd.Shell)
	}
}
