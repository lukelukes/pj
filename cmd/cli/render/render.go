package render

import "time"

type Renderer interface {
	RenderProjectList(view ProjectListView) string
}

type ProjectListView struct {
	Items []ProjectListItem
}

type ProjectListItem struct {
	Name        string
	Path        string
	Description string
	Timestamp   time.Time
}

func (v ProjectListView) IsEmpty() bool {
	return len(v.Items) == 0
}
