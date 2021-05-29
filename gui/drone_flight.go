package main

import (
	"fmt"
	"image/color"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type droneFlightTab struct {
	*baseTab
	useLongJumps   *fltk.CheckButton
	optimizeFor    *fltk.Choice
	solution, keys []lib.Portal
	startPortal    string
	endPortal      string
}

var _ = (*droneFlightTab)(nil)

func newDroneFlightTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *droneFlightTab {
	t := &droneFlightTab{}
	t.baseTab = newBaseTab("Drone Flight", configuration, tileFetcher, t)

	useLongJumpsPack := fltk.NewPack(0, 0, 760, 30)
	useLongJumpsPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.useLongJumps = fltk.NewCheckButton(200, 0, 200, 30, "Use long jumps (key needed)")
	t.useLongJumps.SetValue(true)
	useLongJumpsPack.End()
	t.Add(useLongJumpsPack)

	optimizeForPack := fltk.NewPack(0, 0, 760, 30)
	optimizeForPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.optimizeFor = fltk.NewChoice(200, 0, 200, 30, "Optimize for:")
	t.optimizeFor.Add("Least keys needed", func() {})
	t.optimizeFor.Add("Least jump", func() {})
	t.optimizeFor.SetValue(0)
	optimizeForPack.End()
	t.Add(optimizeForPack)

	t.Add(t.searchSaveCopyPack)
	t.Add(t.progress)
	if t.portalList != nil {
		t.Add(t.portalList)
	}
	t.End()

	return t
}
func (t *droneFlightTab) onReset() {
	t.solution = nil
	t.keys = nil
	t.startPortal = ""
	t.endPortal = ""
}
func (t *droneFlightTab) onSearch() {
	if len(t.portals) < 3 {
		return
	}
	progressFunc := func(val, max int) {
		fltk.Awake(func() {
			t.progress.SetMaximum(float64(max))
			t.progress.SetValue(float64(val))
		})
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
		t.solution, t.keys = lib.LongestDroneFlight(portals, options...)
		if t.mapWindow != nil {
			t.mapWindow.SetPortalPaths([][]lib.Portal{t.solution})
		}
		fltk.Awake(func() {
			if len(t.solution) == 0 {
				t.onSearchDone("No solution found")
			} else {
				distance := t.solution[0].LatLng.Distance(t.solution[len(t.solution)-1].LatLng) * lib.RadiansToMeters
				solutionText := fmt.Sprintf("Flight distance: %.1fm, keys needed: %d", distance, len(t.keys))
				t.onSearchDone(solutionText)
			}
		})
	}()
}

func (t *droneFlightTab) solutionString() string {
	s := fmt.Sprintf("[%s", lib.PolylineFromPortalList(t.solution))
	if len(t.keys) > 0 {
		s += fmt.Sprintf(",%s", lib.MarkersFromPortalList(t.keys))
	}
	return s + "]"
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
func (t *droneFlightTab) portalColor(guid string) color.Color {
	if t.startPortal == guid {
		if _, ok := t.selectedPortals[guid]; !ok {
			return color.NRGBA{0, 255, 0, 128}
		}
		return color.NRGBA{0, 128, 0, 128}
	}
	if t.endPortal == guid {
		if _, ok := t.selectedPortals[guid]; !ok {
			return color.NRGBA{255, 255, 0, 128}
		}
		return color.NRGBA{128, 128, 0, 128}
	}
	return t.baseTab.portalColor(guid)
}

func (t *droneFlightTab) enableSelectedPortals() {
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

func (t *droneFlightTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		t.disabledPortals[guid] = struct{}{}
		if t.startPortal == guid {
			t.startPortal = ""
		}
		if t.endPortal == guid {
			t.endPortal = ""
		}
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
func (t *droneFlightTab) makeSelectedPortalStart() {
	if len(t.selectedPortals) != 1 {
		return
	}

	for guid := range t.selectedPortals {
		t.startPortal = guid
		delete(t.disabledPortals, guid)
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
func (t *droneFlightTab) unmakeSelectedPortalStart() {
	guid := t.startPortal
	t.startPortal = ""
	if t.mapWindow != nil {
		t.mapWindow.SetPortalColor(guid, t.portalColor(guid))
		t.mapWindow.Raise(guid)
	}
	if t.portalList != nil {
		t.portalList.SetPortalLabel(guid, t.portalLabel(guid))
		t.portalList.Redraw()
	}
}
func (t *droneFlightTab) makeSelectedPortalEnd() {
	if len(t.selectedPortals) != 1 {
		return
	}
	for guid := range t.selectedPortals {
		t.endPortal = guid
		delete(t.disabledPortals, guid)
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
func (t *droneFlightTab) unmakeSelectedPortalEnd() {
	guid := t.endPortal
	t.endPortal = ""
	if t.mapWindow != nil {
		t.mapWindow.SetPortalColor(guid, t.portalColor(guid))
		t.mapWindow.Raise(guid)
	}
	if t.portalList != nil {
		t.portalList.SetPortalLabel(guid, t.portalLabel(guid))
		t.portalList.Redraw()
	}
}

func (t *droneFlightTab) contextMenu() *menu {
	var aSelectedGUID string
	var isDisabledSelected, isEnabledSelected, isStartSelected, isEndSelected bool
	numNonStartSelected := 0
	numNonEndSelected := 0
	for guid := range t.selectedPortals {
		aSelectedGUID = guid
		if _, ok := t.disabledPortals[guid]; ok {
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
	if len(t.selectedPortals) > 1 {
		menu.header = fmt.Sprintf("%d portals selected", len(t.selectedPortals))
	} else if len(t.selectedPortals) == 1 {
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
