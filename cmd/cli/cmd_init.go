package main

import "fmt"

type InitCmd struct{}

func (cmd *InitCmd) Run(g *Globals) error { //nolint:unparam // error required by kong interface
	fmt.Fprint(g.Out, shellScript)
	return nil
}

const shellScript = `# pj shell integration
# Add to ~/.bashrc or ~/.zshrc: eval "$(pj init)"

pj() {
    case "$1" in
        cd)
            if [ -z "$2" ]; then
                echo "Usage: pj cd <project>" >&2
                return 1
            fi
            # Capture both stdout and exit code - propagate real errors
            dir="$(command pj show "$2" --path 2>&1)"
            if [ $? -ne 0 ]; then
                echo "pj: $dir" >&2
                return 1
            fi
            if [ ! -d "$dir" ]; then
                echo "pj: path no longer exists: $dir" >&2
                return 1
            fi
            builtin cd -- "$dir"
            ;;
        *)
            command pj "$@"
            ;;
    esac
}
`
