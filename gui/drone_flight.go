package main

import (
	"fmt"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type droneFlightTab struct {
	*baseTab
	useLongJumps *fltk.CheckButton
	optimizeFor  *fltk.Choice
	solution, keys []lib.Portal
}

func NewDroneFlightTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *droneFlightTab {
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
func (t *droneFlightTab) onReset() {}
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
		t.solution, t.keys = lib.LongestDroneFlight(portals, options...)
		if t.mapWindow != nil {
			t.mapWindow.SetPaths([][]lib.Portal{t.solution})
		}
		fltk.Awake(func() {
			distance := t.solution[0].LatLng.Distance(t.solution[len(t.solution)-1].LatLng) * lib.RadiansToMeters
			solutionText := fmt.Sprintf("Flight distance: %.1fm, keys needed: %d", distance, len(t.keys))
			t.onSearchDone(solutionText)
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
func (t *droneFlightTab) onPortalContextMenu(x, y int) {}
