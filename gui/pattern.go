package main

import "image/color"


type menuItem struct {
	label string
	callback func()
}
type menu struct {
	header string
	items []menuItem
}

type pattern interface {
	onSearch()
	portalColor(string) color.Color
	portalLabel(string) string
	solutionString() string
	onReset()
	contextMenu() *menu
}
