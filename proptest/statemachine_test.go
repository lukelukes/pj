package proptest

import (
	"testing"

	"pgregory.net/rapid"
)

func TestProperty_StateMachine_CatalogOperations(t *testing.T) {
	RunWithCatalog(t, func(h *CatalogHarness) {
		checked := NewCheckedCatalog(h.T, h.Catalog)

		h.T.Repeat(map[string]func(*rapid.T){
			"add": func(rt *rapid.T) {
				p := GenProject(rt, h.Dir)
				_ = checked.Add(p)
			},

			"remove": func(rt *rapid.T) {
				ids := checked.Model().IDs()
				if len(ids) == 0 {
					rt.Skip("no projects to remove")
				}
				id := rapid.SampledFrom(ids).Draw(rt, "id")
				_ = checked.Remove(id)
			},

			"get": func(rt *rapid.T) {
				ids := checked.Model().IDs()
				if len(ids) == 0 {
					rt.Skip("no projects")
				}
				id := rapid.SampledFrom(ids).Draw(rt, "id")
				_, _ = checked.Get(id)
			},

			"getByPath": func(rt *rapid.T) {
				ids := checked.Model().IDs()
				if len(ids) == 0 {
					rt.Skip("no projects")
				}
				id := rapid.SampledFrom(ids).Draw(rt, "id")
				realProject, _ := checked.Get(id)
				_, _ = checked.GetByPath(realProject.Path)
			},

			"update": func(rt *rapid.T) {
				ids := checked.Model().IDs()
				if len(ids) == 0 {
					rt.Skip("no projects to update")
				}
				id := rapid.SampledFrom(ids).Draw(rt, "id")
				p, _ := checked.Get(id)
				newStatus := statusGen().Draw(rt, "newStatus")
				p = p.WithStatus(newStatus)
				_ = checked.Update(p)
			},

			"list": func(rt *rapid.T) {
				_ = checked.List()
			},

			"search": func(rt *rapid.T) {
				query := queryGen.Draw(rt, "query")
				_ = checked.Search(query)
			},

			"filter": func(rt *rapid.T) {
				opts := filterOptionsGen().Draw(rt, "filterOpts")
				_ = checked.Filter(opts)
			},
		})
	})
}
