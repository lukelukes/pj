package main

import (
	"fmt"
	"pj/internal/ui"
)

type CreateCmd struct{}

func (cmd *CreateCmd) Run(g *Globals) error { //nolint:unparam // error required by kong interface
	var fields []ui.Field
	output := ui.RenderWizard("Create new project", fields, -1)
	fmt.Fprint(g.Out, output)
	return nil
}
