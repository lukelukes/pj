package main

import (
	"errors"
	"fmt"
)

type CdCmd struct {
	Name string `arg:"" help:"Project name" completion:"pj list -n"`
}

func (cmd *CdCmd) Run(g *Globals) error {
	fmt.Fprintln(g.Out, "The 'cd' command requires shell integration.")
	fmt.Fprintln(g.Out, "Add to your shell config: eval \"$(pj init)\"")
	return errors.New("shell integration required")
}
