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

var _ = (*doubleHerringboneTab)(nil)

func NewDoubleHerringboneTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *doubleHerringboneTab {
	t := &doubleHerringboneTab{}
	t.baseTab = newBaseTab("Double Herringbone", configuration, tileFetcher, t)
	t.End()

	return t
}

func (t *doubleHerringboneTab) onReset() {
	t.basePortals = make(map[string]struct{})
	t.spine0 = nil
	t.spine1 = nil
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
			t.mapWindow.SetPortalPaths([][]lib.Portal{lib.DoubleHerringbonePolyline(t.b0, t.b1, t.spine0, t.spine1)})
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

func (t *doubleHerringboneTab) enableSelectedPortals() {
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

func (t *doubleHerringboneTab) disableSelectedPortals() {
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

func (t *doubleHerringboneTab) makeSelectedPortalsBase() {
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
func (t *doubleHerringboneTab) unmakeSelectedPortalsBase() {
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

func (t *doubleHerringboneTab) contextMenu() *menu {
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
