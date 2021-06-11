package main

import (
	"image/color"

	"github.com/golang/geo/s2"
)

type menuItem struct {
	label    string
	callback func()
}
type menu struct {
	header string
	items  []menuItem
}

type pattern interface {
	onSearch(func(int, int), func())
	portalColor(string) (color.Color, color.Color)
	portalLabel(string) string
	hasSolution() bool
	solutionInfoString() string
	solutionPaths() [][]s2.Point
	solutionDrawToolsString() string
	onReset()
	contextMenu() *menu
}
