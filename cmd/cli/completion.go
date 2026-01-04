package main

import (
	_ "embed"
	"fmt"
	"pj/internal/util"

	"github.com/alecthomas/kong"
	"github.com/miekg/king"
)

//go:embed completions/pj.zsh
var zshCompletion []byte

type CompletionCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell type (bash, zsh, fish)"`
}

func (cmd *CompletionCmd) Run(g *Globals) error {
	switch cmd.Shell {
	case "zsh":
		assert.Success(g.Out.Write(zshCompletion))
	case "bash", "fish":
		cli := CLI{}
		parser, err := kong.New(&cli,
			kong.Name("pj"),
			kong.Description("Project tracker and launcher"),
		)
		if err != nil {
			return err
		}
		node := parser.Model.Node

		if cmd.Shell == "bash" {
			b := &king.Bash{}
			b.Completion(node, "pj")
			assert.Success(g.Out.Write(b.Out()))
		} else {
			f := &king.Fish{}
			f.Completion(node, "pj")
			assert.Success(g.Out.Write(f.Out()))
		}
	default:
		return fmt.Errorf("unsupported shell: %s", cmd.Shell)
	}

	return nil
}
