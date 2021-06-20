package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/lib"
)

type threeCornersTab struct {
	*baseTab
	solution                              []lib.IndexedPortal
	solutionText                          string
	portalsNot0, portalsNot1, portalsNot2 map[string]struct{}
}

var _ pattern = (*threeCornersTab)(nil)

func newThreeCornersTab(portals *Portals) *threeCornersTab {
	t := &threeCornersTab{}
	t.baseTab = newBaseTab("3 Corners", portals, t)
	t.portalsNot0 = make(map[string]struct{})
	t.portalsNot1 = make(map[string]struct{})
	t.portalsNot2 = make(map[string]struct{})
	t.End()

	return t
}

func (t *threeCornersTab) onReset() {
	t.portalsNot0 = make(map[string]struct{})
	t.portalsNot1 = make(map[string]struct{})
	t.portalsNot2 = make(map[string]struct{})
	t.solution = nil
	t.solutionText = ""
}
func (t *threeCornersTab) onSearch(progressFunc func(int, int), onSearchDone func()) {
	portals := t.enabledPortals()
	var portals0, portals1, portals2 []lib.Portal
	for _, portal := range portals {
		if _, ok := t.portalsNot0[portal.Guid]; !ok {
			portals0 = append(portals0, portal)
		}
		if _, ok := t.portalsNot1[portal.Guid]; !ok {
			portals1 = append(portals1, portal)
		}
		if _, ok := t.portalsNot2[portal.Guid]; !ok {
			portals2 = append(portals2, portal)
		}
	}
	go func() {
		solution := lib.LargestThreeCorner(portals0, portals1, portals2, progressFunc)
		fltk.Awake(func() {
			t.solution = solution
			onSearchDone()
		})
	}()
}
func (t *threeCornersTab) hasSolution() bool {
	return len(t.solution) > 0
}
func (t *threeCornersTab) solutionInfoString() string {
	return t.solutionText
}
func (t *threeCornersTab) solutionDrawToolsString() string {
	return lib.ThreeCornersDrawToolsString(t.solution)
}
func (t *threeCornersTab) solutionPaths() [][]s2.Point {
	return [][]s2.Point{portalsToPoints(lib.ThreeCornersPolyline(t.solution))}
}
func (t *threeCornersTab) portalGroups(guid string) []int {
	groups := []int{}
	if _, ok := t.portalsNot0[guid]; !ok {
		groups = append(groups, 0)
	}
	if _, ok := t.portalsNot1[guid]; !ok {
		groups = append(groups, 1)
	}
	if _, ok := t.portalsNot2[guid]; !ok {
		groups = append(groups, 2)
	}
	return groups
}
func (t *threeCornersTab) portalLabel(guid string) string {
	if _, ok := t.portals.disabledPortals[guid]; ok {
		return t.baseTab.portalLabel(guid)
	}
	groups := t.portalGroups(guid)
	groupStrs := []string{}
	for _, group := range groups {
		groupStrs = append(groupStrs, strconv.Itoa(group))
	}
	return "Groups: " + strings.Join(groupStrs, ",")
}
func (t *threeCornersTab) portalColor(guid string) (color.Color, color.Color) {
	if _, ok := t.portals.disabledPortals[guid]; ok {
		return t.baseTab.portalColor(guid)
	}
	groups := t.portalGroups(guid)
	col := color.NRGBA{0, 0, 0, 128}
	for _, group := range groups {
		if group == 0 {
			col.R = 255
		} else if group == 1 {
			col.G = 255
		} else if group == 2 {
			col.B = 255
		} else {
			panic(fmt.Errorf("unexpected group: %d", group))
		}
	}
	if col.R == 255 && col.G == 255 && col.B == 255 {
		return t.baseTab.portalColor(guid)
	}
	return col, t.baseTab.strokeColor(guid)
}

func (t *threeCornersTab) enableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		delete(t.portals.disabledPortals, guid)
	}
}

func (t *threeCornersTab) disableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		t.portals.disabledPortals[guid] = struct{}{}
		//		delete(t.cornerPortals, guid)
	}
}
func (t *threeCornersTab) setSelectedGroup(groups []int) {
	for guid := range t.portals.selectedPortals {
		if len(groups) > 0 {
			delete(t.portals.disabledPortals, guid)
		}
		t.portalsNot0[guid] = struct{}{}
		t.portalsNot1[guid] = struct{}{}
		t.portalsNot2[guid] = struct{}{}
		for _, group := range groups {
			if group == 0 {
				delete(t.portalsNot0, guid)
			} else if group == 1 {
				delete(t.portalsNot1, guid)
			} else if group == 2 {
				delete(t.portalsNot2, guid)
			} else {
				panic(fmt.Errorf("unexpected group %d", group))
			}
		}
	}
}

func (t *threeCornersTab) contextMenu() *menu {
	var aSelectedGUID string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	num0, num1, num2 := 0, 0, 0
	numNot0, numNot1, numNot2 := 0, 0, 0
	for guid := range t.portals.selectedPortals {
		aSelectedGUID = guid
		if _, ok := t.portals.disabledPortals[guid]; ok {
			numSelectedDisabled++
		} else {
			numSelectedEnabled++
		}
		if _, ok := t.portalsNot0[guid]; ok {
			numNot0++
		} else {
			num0++
		}
		if _, ok := t.portalsNot1[guid]; ok {
			numNot1++
		} else {
			num1++
		}
		if _, ok := t.portalsNot2[guid]; ok {
			numNot2++
		} else {
			num2++
		}
	}
	menu := &menu{}
	if len(t.portals.selectedPortals) > 1 {
		menu.header = fmt.Sprintf("%d portals selected", len(t.portals.selectedPortals))
	} else if len(t.portals.selectedPortals) == 1 {
		menu.header = t.portals.portalMap[aSelectedGUID].Name
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
	if numNot0 > 0 || num1 > 0 || num2 > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Set group 0", func() { t.setSelectedGroup([]int{0}) }})
		} else {
			menu.items = append(menu.items, menuItem{"Set all group 0", func() { t.setSelectedGroup([]int{0}) }})
		}
	}
	if numNot1 > 0 || num0 > 0 || num2 > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Set group 1", func() { t.setSelectedGroup([]int{1}) }})
		} else {
			menu.items = append(menu.items, menuItem{"Set all group 0", func() { t.setSelectedGroup([]int{1}) }})
		}
	}
	if numNot2 > 0 || num0 > 0 || num1 > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Set group 2", func() { t.setSelectedGroup([]int{2}) }})
		} else {
			menu.items = append(menu.items, menuItem{"Set all group 2", func() { t.setSelectedGroup([]int{2}) }})
		}
	}
	if numNot0 > 0 || numNot1 > 0 || num2 > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Set group 0,1", func() { t.setSelectedGroup([]int{0, 1}) }})
		} else {
			menu.items = append(menu.items, menuItem{"Set all group 0,1", func() { t.setSelectedGroup([]int{0, 1}) }})
		}
	}
	if numNot0 > 0 || numNot2 > 0 || num1 > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Set group 0,2", func() { t.setSelectedGroup([]int{0, 2}) }})
		} else {
			menu.items = append(menu.items, menuItem{"Set all group 0,2", func() { t.setSelectedGroup([]int{0, 2}) }})
		}
	}
	if numNot1 > 0 || numNot2 > 0 || num0 > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Set group 1,2", func() { t.setSelectedGroup([]int{1, 2}) }})
		} else {
			menu.items = append(menu.items, menuItem{"Set all group 1,2", func() { t.setSelectedGroup([]int{1, 2}) }})
		}
	}
	if numNot0 > 0 || numNot1 > 0 || numNot2 > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Set group 0,1,2", func() { t.setSelectedGroup([]int{0, 1, 2}) }})
		} else {
			menu.items = append(menu.items, menuItem{"Set all group 0,1,2", func() { t.setSelectedGroup([]int{0, 1, 2}) }})
		}
	}
	return menu
}

type indexedGuid struct {
	Index int    `json:"index"`
	Guid  string `json:"guid"`
}
type threeCornersState struct {
	PortalsNot0  []string      `json:"portalsNot0"`
	PortalsNot1  []string      `json:"portalsNot1"`
	PortalsNot2  []string      `json:"portalsNot2"`
	Solution     []indexedGuid `json:"solution"`
	SolutionText string        `json:"solutionText"`
}

func (t *threeCornersTab) state() threeCornersState {
	state := threeCornersState{}
	for portal0GUID := range t.portalsNot0 {
		state.PortalsNot0 = append(state.PortalsNot0, portal0GUID)
	}
	for portal1GUID := range t.portalsNot1 {
		state.PortalsNot1 = append(state.PortalsNot1, portal1GUID)
	}
	for portal2GUID := range t.portalsNot2 {
		state.PortalsNot2 = append(state.PortalsNot2, portal2GUID)
	}
	for _, solutionPortal := range t.solution {
		state.Solution = append(state.Solution,
			indexedGuid{Index: solutionPortal.Index, Guid: solutionPortal.Portal.Guid})
	}
	state.SolutionText = t.solutionText
	return state
}

func (t *threeCornersTab) load(state threeCornersState) error {
	t.portalsNot0 = make(map[string]struct{})
	for _, portal0GUID := range state.PortalsNot0 {
		if _, ok := t.portals.portalMap[portal0GUID]; !ok {
			return fmt.Errorf("unknown three corners portal0 %s", portal0GUID)
		}
		t.portalsNot0[portal0GUID] = struct{}{}
	}
	t.portalsNot1 = make(map[string]struct{})
	for _, portal1GUID := range state.PortalsNot1 {
		if _, ok := t.portals.portalMap[portal1GUID]; !ok {
			return fmt.Errorf("unknown three corners portal1 %s", portal1GUID)
		}
		t.portalsNot1[portal1GUID] = struct{}{}
	}
	t.portalsNot2 = make(map[string]struct{})
	for _, portal2GUID := range state.PortalsNot2 {
		if _, ok := t.portals.portalMap[portal2GUID]; !ok {
			return fmt.Errorf("unknown three corners portal2 %s", portal2GUID)
		}
		t.portalsNot0[portal2GUID] = struct{}{}
	}
	t.solution = nil
	for _, solutionPortal := range state.Solution {
		if portal, ok := t.portals.portalMap[solutionPortal.Guid]; !ok {
			return fmt.Errorf("unknown cobwewb solution portal %s", solutionPortal.Guid)
		} else {
			t.solution = append(t.solution, lib.IndexedPortal{Index: solutionPortal.Index, Portal: portal})
		}
	}
	t.solutionText = state.SolutionText
	return nil
}
