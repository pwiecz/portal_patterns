package main

import (
	"fmt"
	"image/color"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/lib"
)

type droneFlightTab struct {
	*baseTab
	useLongJumps   *fltk.CheckButton
	optimizeFor    *fltk.Choice
	solution, keys []lib.Portal
	solutionText   string
	startPortal    string
	endPortal      string
}

var _ pattern = (*droneFlightTab)(nil)

func newDroneFlightTab(portals *Portals) *droneFlightTab {
	t := &droneFlightTab{}
	t.baseTab = newBaseTab("Drone Flight", portals, t)

	useLongJumpsPack := fltk.NewPack(0, 0, 700, 30)
	useLongJumpsPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.useLongJumps = fltk.NewCheckButton(200, 0, 200, 30, "Use long jumps (key needed)")
	t.useLongJumps.SetValue(true)
	useLongJumpsPack.End()
	t.Add(useLongJumpsPack)

	optimizeForPack := fltk.NewPack(0, 0, 700, 30)
	optimizeForPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.optimizeFor = fltk.NewChoice(200, 0, 200, 30, "Optimize for:")
	t.optimizeFor.Add("Least keys needed", func() {})
	t.optimizeFor.Add("Least jump", func() {})
	t.optimizeFor.SetValue(0)
	optimizeForPack.End()
	t.Add(optimizeForPack)

	t.End()

	return t
}

func (t *droneFlightTab) onReset() {
	t.solution = nil
	t.keys = nil
	t.solutionText = ""
	t.startPortal = ""
	t.endPortal = ""
}
func (t *droneFlightTab) onSearch(progressFunc func(int, int), onSearchDone func()) {
	if len(t.portals.portals) < 3 {
		return
	}
	options := []lib.DroneFlightOption{
		lib.DroneFlightProgressFunc(progressFunc),
		lib.DroneFlightUseLongJumps(t.useLongJumps.Value()),
	}
	switch t.optimizeFor.Value() {
	case 0:
		options = append(options, lib.DroneFlightLeastKeys{})
	case 1:
		options = append(options, lib.DroneFlightLeastJumps{})
	}
	go func() {
		portals := t.enabledPortals()
		for i, portal := range portals {
			if t.startPortal == portal.Guid {
				options = append(options, lib.DroneFlightStartPortalIndex(i))
			}
			if t.endPortal == portal.Guid {
				options = append(options, lib.DroneFlightEndPortalIndex(i))
			}
		}
		solution, keys := lib.LongestDroneFlight(portals, options...)
		fltk.Awake(func() {
			t.solution, t.keys = solution, keys
			if len(t.solution) == 0 {
				t.solutionText = "No flightpath found"
			} else {
				distance := t.solution[0].LatLng.Distance(t.solution[len(t.solution)-1].LatLng) * lib.RadiansToMeters
				t.solutionText = fmt.Sprintf("Flight distance: %.1fm, keys needed: %d", distance, len(t.keys))
			}
			onSearchDone()
		})
	}()
}

func (t *droneFlightTab) hasSolution() bool {
	return len(t.solution) > 0
}
func (t *droneFlightTab) solutionInfoString() string {
	return t.solutionText
}
func (t *droneFlightTab) solutionDrawToolsString() string {
	s := fmt.Sprintf("[%s", lib.PolylineFromPortalList(t.solution))
	if len(t.keys) > 0 {
		s += fmt.Sprintf(",%s", lib.MarkersFromPortalList(t.keys))
	}
	return s + "]"
}
func (t *droneFlightTab) solutionPaths() [][]s2.Point {
	return [][]s2.Point{portalsToPoints(t.solution)}
}
func (t *droneFlightTab) portalLabel(guid string) string {
	if t.startPortal == guid {
		return "Start"
	}
	if t.endPortal == guid {
		return "End"
	}
	return t.baseTab.portalLabel(guid)
}
func (t *droneFlightTab) portalColor(guid string) (color.Color, color.Color) {
	if t.startPortal == guid {
		return color.NRGBA{0, 128, 0, 128}, t.baseTab.strokeColor(guid)
	}
	if t.endPortal == guid {
		return color.NRGBA{128, 128, 0, 128}, t.baseTab.strokeColor(guid)
	}
	return t.baseTab.portalColor(guid)
}

func (t *droneFlightTab) enableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		delete(t.portals.disabledPortals, guid)
	}
}

func (t *droneFlightTab) disableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		t.portals.disabledPortals[guid] = struct{}{}
		if t.startPortal == guid {
			t.startPortal = ""
		}
		if t.endPortal == guid {
			t.endPortal = ""
		}
	}
}
func (t *droneFlightTab) makeSelectedPortalStart() {
	if len(t.portals.selectedPortals) != 1 {
		return
	}

	for guid := range t.portals.selectedPortals {
		t.startPortal = guid
		delete(t.portals.disabledPortals, guid)
	}
}
func (t *droneFlightTab) unmakeSelectedPortalStart() {
	t.startPortal = ""
}
func (t *droneFlightTab) makeSelectedPortalEnd() {
	if len(t.portals.selectedPortals) != 1 {
		return
	}
	for guid := range t.portals.selectedPortals {
		t.endPortal = guid
		delete(t.portals.disabledPortals, guid)
	}
}
func (t *droneFlightTab) unmakeSelectedPortalEnd() {
	t.endPortal = ""
}

func (t *droneFlightTab) contextMenu() *menu {
	var aSelectedGUID string
	var isDisabledSelected, isEnabledSelected, isStartSelected, isEndSelected bool
	numNonStartSelected := 0
	numNonEndSelected := 0
	for guid := range t.portals.selectedPortals {
		aSelectedGUID = guid
		if _, ok := t.portals.disabledPortals[guid]; ok {
			isDisabledSelected = true
		} else {
			isEnabledSelected = true
		}
		if guid == t.startPortal {
			isStartSelected = true
		} else {
			numNonStartSelected++
		}
		if guid == t.endPortal {
			isEndSelected = true
		} else {
			numNonEndSelected++
		}
	}
	menu := &menu{}
	if len(t.portals.selectedPortals) > 1 {
		menu.header = fmt.Sprintf("%d portals selected", len(t.portals.selectedPortals))
	} else if len(t.portals.selectedPortals) == 1 {
		menu.header = t.portalMap[aSelectedGUID].Name
	}
	if isDisabledSelected {
		menu.items = append(menu.items, menuItem{"Enable", t.enableSelectedPortals})
	}
	if isEnabledSelected {
		menu.items = append(menu.items, menuItem{"Disable", t.disableSelectedPortals})
	}
	if numNonStartSelected == 1 && t.startPortal == "" {
		menu.items = append(menu.items, menuItem{"Make start", t.makeSelectedPortalStart})
	}
	if isStartSelected {
		menu.items = append(menu.items, menuItem{"Unmake start", t.unmakeSelectedPortalStart})
	}
	if numNonEndSelected == 1 && t.endPortal == "" {
		menu.items = append(menu.items, menuItem{"Make end", t.makeSelectedPortalEnd})
	}
	if isEndSelected {
		menu.items = append(menu.items, menuItem{"Unmake end", t.unmakeSelectedPortalEnd})
	}
	return menu
}

type droneFlightState struct {
	UseLongJumps bool     `json:"useLongJumps"`
	OptimizeFor  int      `json:"optimizeFor"`
	Solution     []string `json:"solution"`
	Keys         []string `json:"keys"`
	StartPortal  string   `json:"startPortal"`
	EndPortal    string   `json:"endPortal"`
	SolutionText string   `json:"solutionText"`
}

func (t *droneFlightTab) state() droneFlightState {
	state := droneFlightState{
		UseLongJumps: t.useLongJumps.Value(),
		OptimizeFor:  t.optimizeFor.Value(),
		StartPortal:  t.startPortal,
		EndPortal:    t.endPortal,
		SolutionText: t.solutionText,
	}
	for _, solutionPortal := range t.solution {
		state.Solution = append(state.Solution, solutionPortal.Guid)
	}
	for _, keyPortal := range t.keys {
		state.Keys = append(state.Keys, keyPortal.Guid)
	}
	return state
}

func (t *droneFlightTab) load(state droneFlightState) {
	t.useLongJumps.SetValue(state.UseLongJumps)
	t.optimizeFor.SetValue(state.OptimizeFor)
	t.solution = nil
	for _, solutionGUID := range state.Solution {
		if solutionPortal, ok := t.portals.portalMap[solutionGUID]; ok {
			t.solution = append(t.solution, solutionPortal)
		} else {
		}
	}
	for _, keyGUID := range state.Keys {
		if keyPortal, ok := t.portals.portalMap[keyGUID]; ok {
			t.keys = append(t.keys, keyPortal)
		} else {
		}
	}
	if _, ok := t.portals.portalMap[state.StartPortal]; !ok {
	}
	t.startPortal = state.StartPortal
	if _, ok := t.portals.portalMap[state.EndPortal]; !ok {
	}
	t.endPortal = state.EndPortal
	t.solutionText = state.SolutionText
}
