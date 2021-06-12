package main

import (
	"fmt"
	"image/color"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/lib"
)

type cobwebTab struct {
	*baseTab
	solution      []lib.Portal
	solutionText  string
	cornerPortals map[string]struct{}
}

var _ pattern = (*cobwebTab)(nil)

func newCobwebTab(portals *Portals) *cobwebTab {
	t := &cobwebTab{}
	t.baseTab = newBaseTab("Cobweb", portals, t)
	t.cornerPortals = make(map[string]struct{})
	t.End()

	return t
}

func (t *cobwebTab) onReset() {
	t.cornerPortals = make(map[string]struct{})
	t.solution = nil
	t.solutionText = ""
}
func (t *cobwebTab) onSearch(progressFunc func(int, int), onSearchDone func()) {
	go func() {
		portals := t.enabledPortals()
		corners := []int{}
		for i, portal := range portals {
			if _, ok := t.cornerPortals[portal.Guid]; ok {
				corners = append(corners, i)
			}
		}
		solution := lib.LargestCobweb(portals, corners, progressFunc)
		fltk.Awake(func() {
			t.solution = solution
			onSearchDone()
		})
	}()
}
func (t *cobwebTab) hasSolution() bool {
	return len(t.solution) > 0
}
func (t *cobwebTab) solutionInfoString() string {
	return t.solutionText
}
func (t *cobwebTab) solutionDrawToolsString() string {
	return lib.CobwebDrawToolsString(t.solution)
}
func (t *cobwebTab) solutionPaths() [][]s2.Point {
	return [][]s2.Point{portalsToPoints(lib.CobwebPolyline(t.solution))}
}

func (t *cobwebTab) portalLabel(guid string) string {
	if _, ok := t.cornerPortals[guid]; ok {
		return "Corner"
	}
	return t.baseTab.portalLabel(guid)
}
func (t *cobwebTab) portalColor(guid string) (color.Color, color.Color) {
	if _, ok := t.cornerPortals[guid]; ok {
		return color.NRGBA{0, 128, 0, 128}, t.baseTab.strokeColor(guid)
	}
	return t.baseTab.portalColor(guid)
}

func (t *cobwebTab) enableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		delete(t.portals.disabledPortals, guid)
	}
}

func (t *cobwebTab) disableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		t.portals.disabledPortals[guid] = struct{}{}
		delete(t.cornerPortals, guid)
	}
}

func (t *cobwebTab) makeSelectedPortalsCorners() {
	for guid := range t.portals.selectedPortals {
		delete(t.portals.disabledPortals, guid)
		t.cornerPortals[guid] = struct{}{}
	}
}
func (t *cobwebTab) unmakeSelectedPortalsCorners() {
	for guid := range t.portals.selectedPortals {
		delete(t.cornerPortals, guid)
	}
}
func (t *cobwebTab) contextMenu() *menu {
	var aSelectedGUID string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	numSelectedCorner := 0
	numSelectedNotCorner := 0
	for guid := range t.portals.selectedPortals {
		aSelectedGUID = guid
		if _, ok := t.portals.disabledPortals[guid]; ok {
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
	if len(t.portals.selectedPortals) > 1 {
		menu.header = fmt.Sprintf("%d portals selected", len(t.portals.selectedPortals))
	} else if len(t.portals.selectedPortals) == 1 {
		menu.header = t.portalMap[aSelectedGUID].Name
	}
	if numSelectedDisabled > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Enable", t.enableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Enable All", t.enableSelectedPortals})
		}
	}
	if numSelectedEnabled > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Disable", t.disableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Disable All", t.disableSelectedPortals})
		}
	}
	if numSelectedCorner > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Unmake corner", t.unmakeSelectedPortalsCorners})
		} else {
			menu.items = append(menu.items, menuItem{"Unmake all corners", t.unmakeSelectedPortalsCorners})
		}
	}
	if numSelectedNotCorner > 0 && numSelectedNotCorner+len(t.cornerPortals) <= 3 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Make corner", t.makeSelectedPortalsCorners})
		} else {
			menu.items = append(menu.items, menuItem{"Make all corners", t.makeSelectedPortalsCorners})
		}
	}
	return menu
}

type cobwebState struct {
	CornerPortals []string `json:"cornerPortals"`
	Solution      []string `json:"solution"`
	SolutionText  string   `json:"solutionText"`
}

func (t *cobwebTab) state() cobwebState {
	state := cobwebState{}
	for cornerGUID := range t.cornerPortals {
		state.CornerPortals = append(state.CornerPortals, cornerGUID)
	}
	for _, solutionPortal := range t.solution {
		state.Solution = append(state.Solution, solutionPortal.Guid)
	}
	state.SolutionText = t.solutionText
	return state
}

func (t *cobwebTab) load(state cobwebState) error {
	t.cornerPortals = make(map[string]struct{})
	for _, cornerGUID := range state.CornerPortals {
		if _, ok := t.portals.portalMap[cornerGUID]; !ok {
			return fmt.Errorf("unknown cobweb corner portal %s", cornerGUID)
		}
		t.cornerPortals[cornerGUID] = struct{}{}
	}
	t.solution = nil
	for _, solutionGUID := range state.Solution {
		if portal, ok := t.portals.portalMap[solutionGUID]; !ok {
			return fmt.Errorf("unknown cobwewb solution portal %s", solutionGUID)
		} else {
			t.solution = append(t.solution, portal)
		}
	}
	t.solutionText = state.SolutionText
	return nil
}
