package main

import (
	"fmt"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type doubleHerringboneTab struct {
	*baseTab
	b0, b1         lib.Portal
	spine0, spine1 []lib.Portal
	basePortals    map[string]struct{}
}

func NewDoubleHerringboneTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *doubleHerringboneTab {
	t := &doubleHerringboneTab{}
	t.baseTab = newBaseTab("Double Herringbone", configuration, tileFetcher, t)
	t.End()

	return t
}

func (t *doubleHerringboneTab) onReset() {
	t.basePortals = make(map[string]struct{})
}
func (t *doubleHerringboneTab) onSearch() {
	progressFunc := func(val, max int) {
		fltk.Awake(func() {
			t.progress.SetMaximum(float64(max))
			t.progress.SetValue(float64(val))
		})
	}
	go func() {
		portals := t.enabledPortals()
		t.b0, t.b1, t.spine0, t.spine1 = lib.LargestDoubleHerringbone(portals, []int{}, 8, progressFunc)
		if t.mapWindow != nil {
			t.mapWindow.SetPaths([][]lib.Portal{lib.DoubleHerringbonePolyline(t.b0, t.b1, t.spine0, t.spine1)})
		}
		fltk.Awake(func() {
			solutionText := fmt.Sprintf("Solution length: %d + %d", len(t.spine0), len(t.spine1))
			t.onSearchDone(solutionText)
		})
	}()
}

func (t *doubleHerringboneTab) solutionString() string {
	return lib.DoubleHerringboneDrawToolsString(t.b0, t.b1, t.spine0, t.spine1)
}
func (t *doubleHerringboneTab) onPortalContextMenu(x, y int) {}
