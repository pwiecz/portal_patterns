package main

import (
	"fmt"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type HerringboneTab struct {
	*baseTab
	b0, b1      lib.Portal
	spine       []lib.Portal
	basePortals map[string]struct{}
}

func NewHerringboneTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *HerringboneTab {
	t := &HerringboneTab{}
	t.baseTab = newBaseTab("Herringbone", configuration, tileFetcher, t)
	t.End()

	return t
}

func (t *HerringboneTab) onReset() {
	t.basePortals = make(map[string]struct{})
}
func (t *HerringboneTab) onSearch() {
	progressFunc := func(val, max int) {
		fltk.Awake(func() {
			t.progress.SetMaximum(float64(max))
			t.progress.SetValue(float64(val))
		})
	}
	go func() {
		portals := t.enabledPortals()
		t.b0, t.b1, t.spine = lib.LargestHerringbone(portals, []int{}, 8, progressFunc)
		if t.mapWindow != nil {
			t.mapWindow.SetPaths([][]lib.Portal{lib.HerringbonePolyline(t.b0, t.b1, t.spine)})
		}
		fltk.Awake(func() {
			solutionText := fmt.Sprintf("Solution length: %d", len(t.spine))
			t.onSearchDone(solutionText)
		})
	}()
}

func (t *HerringboneTab) solutionString() string {
	return lib.HerringboneDrawToolsString(t.b0, t.b1, t.spine)
}
func (t *HerringboneTab) onPortalContextMenu(x, y int) {}
