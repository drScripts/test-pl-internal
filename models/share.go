package models

type Share struct {
	ID         string
	ProjectId  string
	TableId    string
	ViewId     string
	IsEditable bool
}
