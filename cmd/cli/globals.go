package main

import (
	"io"
	"os"
	"os/exec"
	"pj/cmd/cli/render"
	"pj/internal/catalog"
)

type Globals struct {
	Cat    catalog.Catalog
	Out    io.Writer
	Render render.Renderer
	RunCmd func(name string, args ...string) error
}

func defaultRunCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
