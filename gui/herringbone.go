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

var _ = (*herringboneTab)(nil)

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
		}
		return color.NRGBA{0, 128, 0, 128}
	}
	return t.baseTab.portalColor(guid)
}

func (t *herringboneTab) enableSelectedPortals() {
	for guid := range t.selectedPortals {
		delete(t.basePortals, guid)
		if t.mapWindow != nil {
			t.mapWindow.SetPortalColor(guid, t.pattern.portalColor(guid))
		}
		if t.portalList != nil {
			t.portalList.SetPortalLabel(guid, t.pattern.portalLabel(guid))
		}
	}
	if t.portalList != nil {
		t.portalList.Redraw()
	}
}

func (t *herringboneTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		t.disabledPortals[guid] = struct{}{}
		delete(t.basePortals, guid)
		if t.mapWindow != nil {
			t.mapWindow.SetPortalColor(guid, t.pattern.portalColor(guid))
			t.mapWindow.Lower(guid)
		}
		if t.portalList != nil {
			t.portalList.SetPortalLabel(guid, t.pattern.portalLabel(guid))
		}
	}
	if t.portalList != nil {
		t.portalList.Redraw()
	}
}

func (t *herringboneTab) makeSelectedPortalsBase() {
	for guid := range t.selectedPortals {
		delete(t.disabledPortals, guid)
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

func (t *herringboneTab) contextMenu() *menu {
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
	menu := &menu{}
	if len(t.selectedPortals) > 1 {
		menu.header = fmt.Sprintf("%d portals selected", len(t.selectedPortals))
	} else if len(t.selectedPortals) == 1 {
		menu.header = t.portalMap[aSelectedGuid].Name
	}
	if numSelectedDisabled > 0 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Enable", t.enableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Enable All", t.enableSelectedPortals})
		}
	}
	if numSelectedEnabled > 0 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Disable", t.disableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Disable All", t.disableSelectedPortals})
		}
	}
	if numSelectedBase > 0 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Unmake base", t.unmakeSelectedPortalsBase})
		} else {
			menu.items = append(menu.items, menuItem{"Unmake all base", t.unmakeSelectedPortalsBase})
		}
	}
	if numSelectedNotBase > 0 && numSelectedNotBase+len(t.basePortals) <= 2 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Make base", t.makeSelectedPortalsBase})
		} else {
			menu.items = append(menu.items, menuItem{"Make all base", t.makeSelectedPortalsBase})
		}
	}
	return menu
}
