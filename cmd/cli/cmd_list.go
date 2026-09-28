package main

type ListCmd struct {
	Selector `embed:""`
	Output   Output `short:"o" enum:"table,names,paths,json,tags" default:"table" help:"Output format: table, names, paths, json, tags"`
	Names    bool   `short:"n" hidden:"" help:"Alias for --output names"`
}

func (cmd ListCmd) format() Output {
	if cmd.Names {
		return OutputNames
	}
	return cmd.Output
}

func (cmd *ListCmd) Run(g *Globals) error {
	projects, sort, err := cmd.Select(g.Cat)
	if err != nil {
		return err
	}
	return printProjects(g.Out, g.Render, projects, cmd.format(), sort)
}
