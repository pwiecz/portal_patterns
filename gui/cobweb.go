package main

import (
	"fmt"
	"image/color"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type cobwebTab struct {
	*baseTab
	solution      []lib.Portal
	cornerPortals map[string]struct{}
}

var _ = (*cobwebTab)(nil)

func NewCobwebTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *cobwebTab {
	t := &cobwebTab{}
	t.baseTab = newBaseTab("Cobweb", configuration, tileFetcher, t)
	t.cornerPortals = make(map[string]struct{})
	t.End()

	return t
}

func (t *cobwebTab) onReset() {
	t.cornerPortals = make(map[string]struct{})
	t.solution = nil
}
func (t *cobwebTab) onSearch() {
	progressFunc := func(val, max int) {
		fltk.Awake(func() {
			t.progress.SetMaximum(float64(max))
			t.progress.SetValue(float64(val))
		})
	}
	go func() {
		portals := t.enabledPortals()
		corners := []int{}
		for i, portal := range portals {
			if _, ok := t.cornerPortals[portal.Guid]; ok {
				corners = append(corners, i)
			}
		}
		t.solution = lib.LargestCobweb(portals, corners, progressFunc)
		if t.mapWindow != nil {
			t.mapWindow.SetPortalPaths([][]lib.Portal{lib.CobwebPolyline(t.solution)})
		}
		fltk.Awake(func() {
			solutionText := fmt.Sprintf("Solution length: %d", len(t.solution))
			t.onSearchDone(solutionText)
		})
	}()
}

func (t *cobwebTab) solutionString() string {
	return lib.CobwebDrawToolsString(t.solution)
}

func (t *cobwebTab) portalLabel(guid string) string {
	if _, ok := t.cornerPortals[guid]; ok {
		return "Corner"
	}
	return t.baseTab.portalLabel(guid)
}
func (t *cobwebTab) portalColor(guid string) color.Color {
	if _, ok := t.cornerPortals[guid]; ok {
		if _, ok := t.selectedPortals[guid]; ok {
			return color.NRGBA{0, 255, 0, 128}
		}
		return color.NRGBA{0, 128, 0, 128}
	}
	return t.baseTab.portalColor(guid)
}

func (t *cobwebTab) enableSelectedPortals() {
	for guid := range t.selectedPortals {
		delete(t.disabledPortals, guid)
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

func (t *cobwebTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		t.disabledPortals[guid] = struct{}{}
		delete(t.cornerPortals, guid)
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

func (t *cobwebTab) makeSelectedPortalsCorners() {
	for guid := range t.selectedPortals {
		delete(t.disabledPortals, guid)
		t.cornerPortals[guid] = struct{}{}
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
func (t *cobwebTab) unmakeSelectedPortalsCorners() {
	for guid := range t.selectedPortals {
		delete(t.cornerPortals, guid)
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
func (t *cobwebTab) contextMenu() *menu {
	var aSelectedGuid string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	numSelectedCorner := 0
	numSelectedNotCorner := 0
	for guid := range t.selectedPortals {
		aSelectedGuid = guid
		if _, ok := t.disabledPortals[guid]; ok {
			numSelectedDisabled++
		} else {
			numSelectedEnabled++
		}
		if _, ok := t.cornerPortals[guid]; ok {
			numSelectedCorner++
		} else {
			numSelectedNotCorner++
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
	if numSelectedCorner > 0 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Unmake corner", t.unmakeSelectedPortalsCorners})
		} else {
			menu.items = append(menu.items, menuItem{"Unmake all corners", t.unmakeSelectedPortalsCorners})
		}
	}
	if numSelectedNotCorner > 0 && numSelectedNotCorner+len(t.cornerPortals) <= 3 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Make corner", t.makeSelectedPortalsCorners})
		} else {
			menu.items = append(menu.items, menuItem{"Make all corners", t.makeSelectedPortalsCorners})
		}
	}
	return menu
}
