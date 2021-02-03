package main

import (
	"fmt"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type cobwebTab struct {
	*baseTab
	solution []lib.Portal
}

func NewCobwebTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *cobwebTab {
	t := &cobwebTab{}
	t.baseTab = newBaseTab("Cobweb", configuration, tileFetcher, t)
	t.End()

	return t
}

func (t *cobwebTab) onReset() {}
func (t *cobwebTab) onSearch() {
	progressFunc := func(val, max int) {
		fltk.Awake(func() {
			t.progress.SetMaximum(float64(max))
			t.progress.SetValue(float64(val))
		})
	}
	go func() {
		t.solution = lib.LargestCobweb(t.portals, []int{}, progressFunc)
		if t.mapWindow != nil {
			t.mapWindow.SetPaths([][]lib.Portal{lib.CobwebPolyline(t.solution)})
		}
		fltk.Awake(func() {
			solutionText := fmt.Sprintf("Solution length: %d", len(t.solution))
			t.onSearchDone(solutionText)
		})
	}()
}

func (t *cobwebTab) portalLabel(guid string) string {
	/*	if t.disabledPortals[guid] {
			return "Disabled"
		}
		if t.anchorPortals[guid] {
			return "Anchor"
		}*/
	return "Normal"
}

//func (t *cobwebTab) portalColor(guid string) string {return ""}
func (t *cobwebTab) solutionString() string {
	return lib.CobwebDrawToolsString(t.solution)
}
func (t *cobwebTab) onPortalContextMenu(x, y int) {}
