package wizardtea

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"
)

func TestViewGolden(t *testing.T) {
	adoptStub := stubPreview{adoptNames: map[string]string{"legacy": "~/dev/legacy exists · 12 files"}}
	blockStub := stubPreview{blockNames: map[string]string{"booster": `~/dev/booster is already tracked as "booster"`}}

	tests := map[string]func() *model{
		"form_empty":   func() *model { return newTestModel(t, stubPreview{}) },
		"form_typed":   func() *model { return typeText(t, newTestModel(t, stubPreview{}), "api-gateway") },
		"form_blocked": func() *model { return typeText(t, newTestModel(t, blockStub), "booster") },
		"form_on_location": func() *model {
			return send(t, typeText(t, newTestModel(t, stubPreview{}), "api-gateway"), "tab")
		},
		"form_on_git": func() *model {
			return send(t, typeText(t, newTestModel(t, stubPreview{}), "api-gateway"), "shift+tab")
		},
		"form_git_off": func() *model {
			m := typeText(t, newTestModel(t, stubPreview{}), "api-gateway")
			return send(t, m, "shift+tab", "space")
		},
		"form_all_filled": func() *model {
			m := typeText(t, newTestModel(t, stubPreview{}), "api-gateway")
			m = send(t, m, "enter", "enter")
			return send(t, typeText(t, m, "edge router"), "enter", "enter")
		},
		"confirm_adopt": func() *model {
			return commit(t, typeText(t, newTestModel(t, adoptStub), "legacy"))
		},
	}

	for name, build := range tests {
		t.Run(name, func(t *testing.T) {
			golden.RequireEqual(t, []byte(plainLines(build().render())))
		})
	}
}
