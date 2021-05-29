package main

import (
	"fmt"
	"image/color"
	"path/filepath"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type baseTab struct {
	*fltk.Pack
	configuration      *configuration.Configuration
	add                *fltk.Button
	reset              *fltk.Button
	search             *fltk.Button
	save               *fltk.Button
	copy               *fltk.Button
	solutionLabel      *fltk.Box
	searchSaveCopyPack *fltk.Pack
	progress           *fltk.Progress
	portalList         *portalList
	tileFetcher        *osm.MapTiles
	mapWindow          *MapWindow
	portals            []lib.Portal
	portalMap          map[string]lib.Portal
	selectedPortals    map[string]struct{}
	disabledPortals    map[string]struct{}
	pattern            pattern
	name               string
}

func newBaseTab(name string, configuration *configuration.Configuration, tileFetcher *osm.MapTiles, pattern pattern) *baseTab {
	t := &baseTab{
		selectedPortals: make(map[string]struct{}),
		disabledPortals: make(map[string]struct{}),
	}
	t.Pack = fltk.NewPack(20, 40, 760, 540, name)
	t.SetType(fltk.VERTICAL)
	t.SetSpacing(5)

	t.name = name
	t.configuration = configuration
	t.tileFetcher = tileFetcher
	t.pattern = pattern

	fltk.NewBox(fltk.NO_BOX, 0, 0, 760, 5) // padding at the top
	addResetPack := fltk.NewPack(0, 0, 200, 30)
	addResetPack.SetType(fltk.HORIZONTAL)
	addResetPack.SetSpacing(5)
	t.add = fltk.NewButton(0, 0, 101, 30, "Add portals")
	t.add.SetCallback(func() { t.onAddPortalsPressed() })
	t.reset = fltk.NewButton(0, 0, 113, 30, "Reset portals")
	t.reset.Deactivate()
	t.reset.SetCallback(func() { t.onResetPortalsPressed() })
	addResetPack.End()

	t.searchSaveCopyPack = fltk.NewPack(0, 0, 740, 30)
	t.searchSaveCopyPack.SetType(fltk.HORIZONTAL)
	t.searchSaveCopyPack.SetSpacing(5)
	t.search = fltk.NewButton(0, 0, 70, 30, "Search")
	t.search.Deactivate()
	t.search.SetCallback(func() { t.onSearchPressed() })
	t.save = fltk.NewButton(0, 0, 117, 30, "Save Solution")
	t.save.Deactivate()
	t.save.SetCallback(func() { t.onSavePressed() })
	t.copy = fltk.NewButton(0, 0, 147, 30, "Copy to Clipboard")
	t.copy.Deactivate()
	t.copy.SetCallback(func() { t.onCopyPressed() })
	t.solutionLabel = fltk.NewBox(fltk.NO_BOX, 0, 0, 300, 30)
	t.solutionLabel.SetAlign(fltk.ALIGN_INSIDE)
	t.searchSaveCopyPack.Add(t.solutionLabel)
	t.searchSaveCopyPack.Resizable(t.solutionLabel)
	t.searchSaveCopyPack.End()

	t.progress = fltk.NewProgress(0, 0, 740, 30)
	t.progress.SetSelectionColor(0x4444ff00)

	t.portalList = newPortalList(0, 0, 760, 540)
	t.portalList.SetSelectionChangeCallback(func() { t.OnSelectionChanged(t.portalList.selectedPortals) })
	t.portalList.SetContextMenuCallback(t.onContextMenu)
	t.Resizable(t.portalList)

	return t
}

func (t *baseTab) onSearchPressed() {
	t.add.Deactivate()
	t.reset.Deactivate()
	t.search.Deactivate()
	t.save.Deactivate()
	t.copy.Deactivate()
	if t.portalList != nil {
		t.portalList.Deactivate()
	}
	t.pattern.onSearch()
}
func (t *baseTab) onSearchDone(solutionText string) {
	t.add.Activate()
	t.reset.Activate()
	t.search.Activate()
	t.save.Activate()
	t.copy.Activate()
	if t.portalList != nil {
		t.portalList.Activate()
	}
	t.solutionLabel.SetLabel(solutionText)
}

func (t *baseTab) onAddPortalsPressed() {
	fileChooser := fltk.NewFileChooser(t.configuration.PortalsDirectory, "JSON files (*.json)\tCSV files (*.csv)", fltk.SINGLE, "Select portals file")
	fileChooser.SetCallback(
		func() {
			if !fileChooser.Shown() {
				t.onPortalsFileSelected(fileChooser)
			}
		})
	fileChooser.Show()
}

func (t *baseTab) onPortalsFileSelected(fileChooser *fltk.FileChooser) {
	selection := fileChooser.Selection()
	fileChooser.Destroy()
	if len(selection) != 1 {
		return
	}
	filename := selection[0]
	portalsDir, _ := filepath.Split(filename)
	t.configuration.PortalsDirectory = portalsDir
	if t.mapWindow == nil {
		t.mapWindow = NewMapWindow("Homogeneous", t.tileFetcher)
		t.mapWindow.SetSelectionChangeCallback(t.OnSelectionChanged)
		t.mapWindow.SetAddedToSelectionCallback(func(selection map[string]struct{}) {
			selectionCopy := make(map[string]struct{})
			for guid := range t.selectedPortals {
				selectionCopy[guid] = struct{}{}
			}
			for guid := range selection {
				selectionCopy[guid] = struct{}{}
			}
			t.OnSelectionChanged(selectionCopy)
		})
		t.mapWindow.SetRightClickCallback(func(guid string, x, y int) {
			if guid != "" {
				if _, ok := t.selectedPortals[guid]; !ok {
					selection := make(map[string]struct{})
					selection[guid] = struct{}{}
					t.OnSelectionChanged(selection)
				}
			}
			t.onContextMenu(x, y)
		})
		t.mapWindow.SetWindowClosedCallback(t.onMapWindowClosed)
	} else {
		t.mapWindow.Show()
	}
	portals, _ := lib.ParseFile(filename)
	t.addPortals(portals)
	if t.portalList != nil {
		t.portalList.SetPortals(t.portals)
	}
	if t.mapWindow != nil {
		t.mapWindow.SetPortals(t.portals)
	}
	if len(t.portals) >= 3 {
		t.search.Activate()
	} else {
		t.search.Deactivate()
	}
	if len(t.portals) > 0 {
		t.reset.Activate()
	}
}

func (t *baseTab) onResetPortalsPressed() {
	t.portals = t.portals[:0]
	t.selectedPortals = make(map[string]struct{})
	t.disabledPortals = make(map[string]struct{})
	t.reset.Deactivate()
	t.search.Deactivate()
	t.save.Deactivate()
	t.copy.Deactivate()
	if t.mapWindow != nil {
		t.mapWindow.Destroy()
		t.mapWindow = nil
	}
	if t.portalList != nil {
		t.portalList.SetPortals([]lib.Portal{})
	}
	t.solutionLabel.SetLabel("")
	t.pattern.onReset()
}

func (t *baseTab) addPortals(portals []lib.Portal) {
	portalMap := make(map[string]lib.Portal)
	for _, portal := range t.portals {
		portalMap[portal.Guid] = portal
	}
	newPortals := ([]lib.Portal)(nil)
	for _, portal := range portals {
		if existing, ok := portalMap[portal.Guid]; ok {
			if existing.LatLng.Lat != portal.LatLng.Lat ||
				existing.LatLng.Lng != portal.LatLng.Lng {
				if existing.Name == portal.Name {
					fltk.MessageBox("Conflicting portals", "Portal with guid \""+portal.Guid+"\" already loaded with different location\n"+
						portal.Name+"\n"+portal.LatLng.String()+" vs "+existing.LatLng.String())
					return
				}
				fltk.MessageBox("Conflicting portals", "Portal with guid \""+portal.Guid+"\" already loaded with different name and location\n"+
					portal.Name+" vs "+existing.Name+"\n"+portal.LatLng.String()+" vs "+existing.LatLng.String())
				return
			}
		} else {
			portalMap[portal.Guid] = portal
			newPortals = append(newPortals, portal)
		}
	}

	hull := s2.NewConvexHullQuery()
	for _, p := range portalMap {
		hull.AddPoint(s2.PointFromLatLng(p.LatLng))
	}
	if hull.CapBound().Radius().Radians() >= 1. {
		fltk.MessageBox("Too distant portals", "Distances between portals are too large")
		return
	}
	t.portals = append(t.portals, newPortals...)
	t.portalMap = make(map[string]lib.Portal)
	for _, portal := range newPortals {
		t.portalMap[portal.Guid] = portal
	}
}

func (t *baseTab) onSavePressed() {}
func (t *baseTab) onCopyPressed() {}
func stringSetsAreTheSame(map1 map[string]struct{}, map2 map[string]struct{}) bool {
	for s := range map1 {
		if _, ok := map2[s]; !ok {
			return false
		}
	}
	for s := range map2 {
		if _, ok := map1[s]; !ok {
			return false
		}
	}
	return true
}
func stringSetCopy(set map[string]struct{}) map[string]struct{} {
	setCopy := make(map[string]struct{})
	for s := range set {
		setCopy[s] = struct{}{}
	}
	return setCopy
}

func (t *baseTab) portalColor(guid string) color.Color {
	_, isSelected := t.selectedPortals[guid]
	_, isDisabled := t.disabledPortals[guid]
	if isDisabled {
		if isSelected {
			return color.NRGBA{64, 64, 64, 128}
		}
		return color.NRGBA{128, 128, 128, 128}
	}
	if isSelected {
		return color.NRGBA{128, 0, 0, 128}
	}
	return color.NRGBA{255, 128, 0, 128}
}

func (t *baseTab) portalLabel(guid string) string {
	_, isDisabled := t.disabledPortals[guid]
	if isDisabled {
		return "Disabled"
	}
	return "Normal"
}

func (t *baseTab) OnSelectionChanged(selectedPortals map[string]struct{}) {
	if stringSetsAreTheSame(selectedPortals, t.selectedPortals) {
		return
	}
	var selectedBefore map[string]struct{}
	t.selectedPortals, selectedBefore = stringSetCopy(selectedPortals), t.selectedPortals
	if t.portalList != nil {
		t.portalList.SetSelectedPortals(t.selectedPortals)
	}
	if t.mapWindow != nil {
		for guid := range selectedBefore {
			t.mapWindow.SetPortalColor(guid, t.pattern.portalColor(guid))
		}
		for guid := range t.selectedPortals {
			t.mapWindow.SetPortalColor(guid, t.pattern.portalColor(guid))
			t.mapWindow.Raise(guid)
		}
	}
	if len(t.selectedPortals) == 1 {
		if t.mapWindow != nil {
			//t.mapWindow.ScrollToPortal(selection[0])
		}
		if t.mapWindow != nil {
			//t.mapWindow.ScrollToPortal(selection[0])
		}
	}
}
func (t *baseTab) OnPortalSelected(guid string) {
	selection := make(map[string]struct{})
	selection[guid] = struct{}{}
	t.OnSelectionChanged(selection)
	if t.portalList != nil {
		//t.portalList.ScrollToPortal(guid)
	}
	if t.mapWindow != nil {
		//t.solutionMap.ScrollToPortal(guid)
	}
}

func (t *baseTab) enabledPortals() []lib.Portal {
	portals := []lib.Portal{}
	for _, portal := range t.portals {
		if _, ok := t.disabledPortals[portal.Guid]; !ok {
			portals = append(portals, portal)
		}
	}
	return portals
}

func (t *baseTab) onContextMenu(x, y int) {
	menu := t.pattern.contextMenu()
	if menu == nil || len(menu.items) == 0 {
		return
	}
	mb := fltk.NewMenuButton(x, y, 100, 100, menu.header)
	mb.SetCallback(func() { fmt.Println("menu callback") })
	mb.SetType(fltk.POPUP3)
	for _, item := range menu.items {
		mb.Add(item.label, item.callback)
	}
	mb.Popup()
	mb.Destroy()
}
func (t *baseTab) onMapWindowClosed() {
	t.mapWindow.Destroy()
	t.mapWindow = nil
}
