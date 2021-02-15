package main

import (
	"fmt"
	"image/color"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type herringboneTab struct {
	*baseTab
	b0, b1      lib.Portal
	spine       []lib.Portal
	basePortals map[string]struct{}
}

func NewHerringboneTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *herringboneTab {
	t := &herringboneTab{
		basePortals: make(map[string]struct{}),
	}
	t.baseTab = newBaseTab("Herringbone", configuration, tileFetcher, t)
	t.End()

	return t
}

func (t *herringboneTab) onReset() {
	t.basePortals = make(map[string]struct{})
}
func (t *herringboneTab) onSearch() {
	progressFunc := func(val, max int) {
		fltk.Awake(func() {
			t.progress.SetMaximum(float64(max))
			t.progress.SetValue(float64(val))
		})
	}
	go func() {
		portals := t.enabledPortals()
		base := []int{}
		for i, portal := range portals {
			if _, ok := t.basePortals[portal.Guid]; ok {
				base = append(base, i)
			}
		}
		t.b0, t.b1, t.spine = lib.LargestHerringbone(portals, base, 8, progressFunc)
		if t.mapWindow != nil {
			t.mapWindow.SetPaths([][]lib.Portal{lib.HerringbonePolyline(t.b0, t.b1, t.spine)})
		}
		fltk.Awake(func() {
			solutionText := fmt.Sprintf("Solution length: %d", len(t.spine))
			t.onSearchDone(solutionText)
		})
	}()
}

func (t *herringboneTab) solutionString() string {
	return lib.HerringboneDrawToolsString(t.b0, t.b1, t.spine)
}

func (t *herringboneTab) portalLabel(guid string) string {
	if _, ok := t.basePortals[guid]; ok {
		return "Base"
	}
	return t.baseTab.portalLabel(guid)
}
func (t *herringboneTab) portalColor(guid string) color.Color {
	if _, ok := t.basePortals[guid]; ok {
		if _, ok := t.selectedPortals[guid]; ok {
			return color.NRGBA{0, 255, 0, 128}
		} else {
			return color.NRGBA{0, 128, 0, 128}
		}
	}
	return t.baseTab.portalColor(guid)
}
func (t *herringboneTab) makeSelectedPortalsBase() {
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			continue
		}
		t.basePortals[guid] = struct{}{}
		if t.mapWindow != nil {
			t.mapWindow.SetPortalColor(guid, t.portalColor(guid))
			t.mapWindow.Raise(guid)
		}
		if t.portalList != nil {
			t.portalList.SetPortalLabel(guid, t.portalLabel(guid))
		}
	}
	if t.portalList != nil {
		t.portalList.Redraw()
	}
}
func (t *herringboneTab) unmakeSelectedPortalsBase() {
	for guid := range t.selectedPortals {
		delete(t.basePortals, guid)
		if t.mapWindow != nil {
			t.mapWindow.SetPortalColor(guid, t.portalColor(guid))
			t.mapWindow.Raise(guid)
		}
		if t.portalList != nil {
			t.portalList.SetPortalLabel(guid, t.portalLabel(guid))
		}
	}
	if t.portalList != nil {
		t.portalList.Redraw()
	}
}

func (t *herringboneTab) onPortalContextMenu(x, y int) {
	var aSelectedGuid string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	numSelectedBase := 0
	numSelectedNotBase := 0
	for guid := range t.selectedPortals {
		aSelectedGuid = guid
		if _, ok := t.disabledPortals[guid]; ok {
			numSelectedDisabled++
		} else {
			numSelectedEnabled++
		}
		if _, ok := t.basePortals[guid]; ok {
			numSelectedBase++
		} else {
			numSelectedNotBase++
		}
	}
	menuHeader := fmt.Sprintf("%d portals selected", len(t.selectedPortals))
	if len(t.selectedPortals) == 1 {
		menuHeader = t.portalMap[aSelectedGuid].Name
	}
	mb := fltk.NewMenuButton(x, y, 100, 100, menuHeader)
	mb.SetCallback(func() { fmt.Println("menu Callback") })
	mb.SetType(fltk.POPUP3)
	if numSelectedDisabled > 0 {
		if len(t.selectedPortals) == 1 {
			mb.Add("Enable", func() { t.enableSelectedPortals() })
		} else {
			mb.Add("Enable All", func() { t.enableSelectedPortals() })
		}
	}
	if numSelectedEnabled > 0 {
		if len(t.selectedPortals) == 1 {
			mb.Add("Disable", func() { t.disableSelectedPortals() })
		} else {
			mb.Add("Disable All", func() { t.disableSelectedPortals() })
		}
	}
	if numSelectedBase > 0 {
		if len(t.selectedPortals) == 1 {
			mb.Add("Unmake base", func() { t.unmakeSelectedPortalsBase() })
		} else {
			mb.Add("Unmake all base", func() { t.unmakeSelectedPortalsBase() })
		}
	}
	if numSelectedNotBase > 0 && numSelectedNotBase+len(t.basePortals) <= 2 {
		if len(t.selectedPortals) == 1 {
			mb.Add("Make base", func() { t.makeSelectedPortalsBase() })
		} else {
			mb.Add("Make all base", func() { t.makeSelectedPortalsBase() })
		}
	}
	mb.Popup()
	mb.Destroy()
}
