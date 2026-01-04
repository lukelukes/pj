package main

import (
	"pj/internal/util"

	"github.com/alecthomas/kong"
	"github.com/miekg/king"
)

type CompletionCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish" help:"Shell type (bash, zsh, fish)"`
}

func (cmd *CompletionCmd) Run(g *Globals) error {
	cli := CLI{}
	parser, err := kong.New(&cli,
		kong.Name("pj"),
		kong.Description("Project tracker and launcher"),
	)
	if err != nil {
		return err
	}

	node := parser.Model.Node

	switch cmd.Shell {
	case "bash":
		b := &king.Bash{}
		b.Completion(node, "pj")
		assert.Success(g.Out.Write(b.Out()))
	case "zsh":
		z := &king.Zsh{}
		z.Completion(node, "pj")
		assert.Success(g.Out.Write(z.Out()))
	case "fish":
		f := &king.Fish{}
		f.Completion(node, "pj")
		assert.Success(g.Out.Write(f.Out()))
	}

	return nil
}
