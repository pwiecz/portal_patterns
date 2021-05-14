package main

import (
	"fmt"
	"image/color"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type droneFlightTab struct {
	*baseTab
	useLongJumps *widget.Check
	optimizeFor  *widget.Select
	solution     []lib.Portal
	keys         []lib.Portal
	startPortal  string
	endPortal    string
}

var _ pattern = (*droneFlightTab)(nil)

func NewDroneFlightTab(app fyne.App, parent fyne.Window, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *container.TabItem {
	t := &droneFlightTab{}
	t.baseTab = NewBaseTab(app, parent, "Drone Flight", t, conf, tileFetcher)
	useLongJumpsLabel := widget.NewLabel("Use long jumps (key needed): ")
	t.useLongJumps = widget.NewCheck("", func(bool) {})
	optimizeForLabel := widget.NewLabel("Optimize for: ")
	t.optimizeFor = widget.NewSelect([]string{"Least keys needed", "Least jumps"}, func(string) {})
	t.optimizeFor.SetSelectedIndex(0)
	content := container.New(
		layout.NewGridLayout(2),
		useLongJumpsLabel, t.useLongJumps,
		optimizeForLabel, t.optimizeFor)
	topContent := container.NewVBox(
		container.NewHBox(t.add, t.reset),
		content,
		container.NewHBox(t.find, t.save, t.copy, t.solutionLabel),
		t.progress,
	)
	return container.NewTabItem("Drone Flight",
		container.New(
			layout.NewBorderLayout(topContent, nil, nil, nil),
			topContent))
}

func (t *droneFlightTab) onReset() {
	t.startPortal = ""
	t.endPortal = ""
}

func (t *droneFlightTab) portalLabel(guid string) string {
	if _, ok := t.disabledPortals[guid]; ok {
		return "Disabled"
	}
	if t.startPortal == guid {
		return "Start"
	}
	if t.endPortal == guid {
		return "End"
	}
	return "Normal"
}

func (t *droneFlightTab) portalColor(guid string) color.NRGBA {
	if _, ok := t.disabledPortals[guid]; ok {
		if _, ok := t.selectedPortals[guid]; !ok {
			return color.NRGBA{128, 128, 128, 255}
		}
		return color.NRGBA{64, 64, 64, 255}
	}
	if t.startPortal == guid {
		if _, ok := t.selectedPortals[guid]; !ok {
			return color.NRGBA{0, 255, 0, 255}
		}
		return color.NRGBA{0, 128, 0, 255}
	}
	if t.endPortal == guid {
		if _, ok := t.selectedPortals[guid]; !ok {
			return color.NRGBA{255, 255, 0, 255}
		}
		return color.NRGBA{128, 128, 0, 255}
	}
	if _, ok := t.selectedPortals[guid]; !ok {
		return color.NRGBA{255, 178, 0, 255}
	}
	return color.NRGBA{255, 0, 0, 255}
}

func (t *droneFlightTab) search() {
	if len(t.portals) < 3 {
		return
	}

	portals := []lib.Portal{}
	options := []lib.DroneFlightOption{lib.DroneFlightNumWorkers(runtime.GOMAXPROCS(0))}
	for _, portal := range t.portals {
		if _, ok := t.disabledPortals[portal.Guid]; !ok {
			portals = append(portals, portal)
			if t.startPortal == portal.Guid {
				options = append(options, lib.DroneFlightStartPortalIndex(len(portals)-1))
			}
			if t.endPortal == portal.Guid {
				options = append(options, lib.DroneFlightEndPortalIndex(len(portals)-1))
			}
		}
	}
	options = append(options, lib.DroneFlightUseLongJumps(t.useLongJumps.Checked))
	options = append(options, lib.DroneFlightProgressFunc(
		func(val int, max int) { t.onProgress(val, max) }))
	if t.optimizeFor.SelectedIndex() == 1 {
		options = append(options, lib.DroneFlightLeastJumps{})
	}

	t.add.Disable()
	t.reset.Disable()
	t.useLongJumps.Disable()
	t.optimizeFor.Disable()
	t.find.Disable()
	t.save.Disable()
	t.copy.Disable()

	t.solutionLabel.SetText("")
	t.solution, t.keys = lib.LongestDroneFlight(portals, options...)
	if t.solutionMap != nil {
		t.solutionMap.SetSolution([][]lib.Portal{t.solution})
	}
	distance := t.solution[0].LatLng.Distance(t.solution[len(t.solution)-1].LatLng) * lib.RadiansToMeters
	solutionText := fmt.Sprintf("Flight distance: %.1fm, keys needed: %d", distance, len(t.keys))
	t.solutionLabel.SetText(solutionText)
	t.add.Enable()
	t.reset.Enable()
	t.useLongJumps.Enable()
	t.optimizeFor.Enable()
	t.find.Enable()
	t.save.Enable()
	t.copy.Enable()
}

func (t *droneFlightTab) solutionString() string {
	s := fmt.Sprintf("[%s", lib.PolylineFromPortalList(t.solution))
	if len(t.keys) > 0 {
		s += fmt.Sprintf(",%s", lib.MarkersFromPortalList(t.keys))
	}
	return s + "]"
}

func (t *droneFlightTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			continue
		}
		t.disabledPortals[guid] = struct{}{}
		if t.startPortal == guid {
			t.startPortal = ""
		}
		if t.endPortal == guid {
			t.endPortal = ""
		}
		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}
func (t *droneFlightTab) makeSelectedPortalStart() {
	if len(t.selectedPortals) != 1 {
		return
	}

	for guid := range t.selectedPortals {
		t.startPortal = guid
		delete(t.disabledPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
func (t *droneFlightTab) unmakeSelectedPortalStart() {
	for guid := range t.selectedPortals {
		if t.startPortal == guid {
			t.startPortal = ""
			t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
			return
		}
	}
}
func (t *droneFlightTab) makeSelectedPortalEnd() {
	if len(t.selectedPortals) != 1 {
		return
	}
	for guid := range t.selectedPortals {
		t.endPortal = guid
		delete(t.disabledPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
func (t *droneFlightTab) unmakeSelectedPortalEnd() {
	for guid := range t.selectedPortals {
		if t.endPortal == guid {
			t.endPortal = ""
			t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
			return
		}
	}
}

func (t *droneFlightTab) onContextMenu(x, y float32) {
	menuItems := []*fyne.MenuItem{}
	var isDisabledSelected, isEnabledSelected, isStartSelected, isEndSelected bool
	numNonStartSelected := 0
	numNonEndSelected := 0
	for guid, _ := range t.selectedPortals {
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
	if isDisabledSelected {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Enable", t.enableSelectedPortals))
	}
	if isEnabledSelected {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Disable", t.disableSelectedPortals))
	}
	if numNonStartSelected == 1 && t.startPortal == "" {
		menuItems = append(menuItems, 
			fyne.NewMenuItem("Make start", t.makeSelectedPortalStart))
	}
	if isStartSelected {
		menuItems = append(menuItems, 
			fyne.NewMenuItem("Unmake start", t.unmakeSelectedPortalStart))
	}
	if numNonEndSelected == 1 && t.endPortal == "" {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Make end", t.makeSelectedPortalEnd))
	}
	if isEndSelected {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Unmake end", t.unmakeSelectedPortalEnd))
	}
	if len(menuItems) == 0 {
		return
	}
	menu := fyne.NewMenu("", menuItems...)
	menu.Items = menuItems
	widget.ShowPopUpMenuAtPosition(menu, t.app.Driver().CanvasForObject(t.solutionMap),
		fyne.NewPos(x, y))
}
