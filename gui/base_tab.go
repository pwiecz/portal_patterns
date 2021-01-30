package main

import (
	"fmt"
	"path/filepath"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type pattern interface {
	onSearch()
	portalColor(string) string
	portalLabel(string) string
	solutionString() string
	onReset()
	onPortalContextMenu(guid string, x, y int)
}

type baseTab struct {
	configuration      *configuration.Configuration
	add                *fltk.Button
	reset              *fltk.Button
	search             *fltk.Button
	save               *fltk.Button
	copy               *fltk.Button
	solutionLabel      *fltk.Box
	searchSaveCopyPack *fltk.Pack
	progress           *fltk.Progress
	portalList         *fltk.TableRow
	tileFetcher        *osm.MapTiles
	mapWindow          *MapWindow
	portals            []lib.Portal
	selectedPortals    map[string]struct{}
	disabledPortals    map[string]struct{}
	pattern            pattern
	name               string
}

func newBaseTab(name string, configuration *configuration.Configuration, tileFetcher *osm.MapTiles, pattern pattern) *baseTab {
	t := &baseTab{}
	t.name = name
	t.configuration = configuration
	t.tileFetcher = tileFetcher
	t.pattern = pattern

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

	t.portalList = fltk.NewTableRow(0, 0, 100, 540)
	t.portalList.SetDrawCellCallback(func(context fltk.TableContext, r, c, x, y, w, h int) {
		t.PortalListDrawCallback(context, r, c, x, y, w, h)
	})
	t.progress = fltk.NewProgress(0, 0, 740, 30)
	t.progress.SetSelectionColor(0x4444ff00)
	t.portalList.EnableColumnHeaders()
	t.portalList.AllowColumnResizing()
	t.portalList.SetColumnWidth(0, 200)
//	t.portalList.SetEventHandler(func(event fltk.Event) bool {
//		return t.handlePortalListEvent(event)
//	})
	t.portalList.SetType(fltk.SelectMulti)

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
	filename, ok := fltk.ChooseFile(
		"Select portals file",
		"JSON files (*.json)\tCSV files (*.csv)", t.configuration.PortalsDirectory, false)
	if !ok {
		return
	}
	portalsDir, _ := filepath.Split(filename)
	t.configuration.PortalsDirectory = portalsDir
	if t.mapWindow == nil {
		t.mapWindow = NewMapWindow("Homogeneous", t.tileFetcher)
	} else {
		t.mapWindow.Show()
	}
	portals, _ := lib.ParseFile(filename)
	t.addPortals(portals)
	t.portals = portals
	if t.portalList != nil {
		t.portalList.SetRowCount(len(t.portals))
		t.portalList.SetColumnCount(2)
	}
	if t.mapWindow != nil {
		t.mapWindow.SetPortals(t.portals)
	}
	if len(t.portals) > 0 {
		t.search.Activate()
	} else {
		t.search.Deactivate()
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
		//t.mapWindow.Clear()
		t.mapWindow.Hide()
	}
	if t.portalList != nil {
		t.portalList.SetRowCount(0)
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
	if len(t.portals) > 0 {
		t.reset.Activate()
	}
	if len(t.portals) >= 3 {
		t.search.Activate()
	}
	if t.mapWindow == nil {
		t.mapWindow = NewMapWindow(t.name, t.tileFetcher)
		//		t.solutionMap.OnPortalLeftClick(func(guid string) {
		//			t.OnPortalSelected(guid)
		//		})
		//		t.solutionMap.OnPortalRightClick(func(guid string, x, y int) {
		//			t.pattern.onPortalContextMenu(guid, x, y)
		//		})
		//		t.solutionMap.ShowNormal()
		//		tk.Update()
	} else {
		t.mapWindow.Show()
	}
	t.mapWindow.SetPortals(t.portals)
	if t.portalList != nil {
		t.portalList.SetRowCount(len(t.portals))
	}
	//	for _, portal := range t.portals {
	//		t.portalStateChanged(portal.Guid)
	//	}
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

func (t *baseTab) OnSelectionChanged(selection []string) {
	selectionMap := make(map[string]struct{})
	for _, guid := range selection {
		selectionMap[guid] = struct{}{}
	}
	if stringSetsAreTheSame(selectionMap, t.selectedPortals) {
		return
	}
	//	oldSelected := t.selectedPortals
	t.selectedPortals = selectionMap
	if t.portalList != nil {
		//t.portalList.SetSelectedPortals(t.selectedPortals)
	}
	if t.mapWindow != nil {
		//for guid := range oldSelected {
		//t.mapWindow.SetPortalColor(guid, t.pattern.portalColor(guid))
		//}
		//for _, guid := range selection {
		//t.mapWindow.SetPortalColor(guid, t.pattern.portalColor(guid))
		//t.mapWindow.RaisePortal(guid)
		//}
	}
	if len(selection) == 1 {
		if t.mapWindow != nil {
			//t.mapWindow.ScrollToPortal(selection[0])
		}
		if t.mapWindow != nil {
			//t.mapWindow.ScrollToPortal(selection[0])
		}
	}
}
func (t *baseTab) OnPortalSelected(guid string) {
	t.OnSelectionChanged([]string{guid})
	if t.portalList != nil {
		//t.portalList.ScrollToPortal(guid)
	}
	if t.mapWindow != nil {
		//t.solutionMap.ScrollToPortal(guid)
	}
}
func (t *baseTab) portalStateChanged(guid string) {
	if t.portalList != nil {
		//t.portalList.SetPortalState(guid, t.pattern.portalLabel(guid))
	}
	if t.mapWindow != nil {
		//t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}

func (t *baseTab) PortalListDrawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
	switch context {
	case fltk.ContextCell:
		if row >= len(t.portals) {
			return
		}
		background := uint(0xffffffff)
		if t.portalList.IsRowSelected(row) {
			background = t.portalList.SelectionColor()
		}
		fltk.DrawBox(fltk.THIN_UP_BOX, x, y, w, h, background)
		fltk.Color(0x00000000)
		if column == 0 {
			fltk.Draw(t.portals[row].Name, x, y, w, h, fltk.ALIGN_LEFT)
		} else if column == 1 {
			fltk.Draw(t.pattern.portalLabel(t.portals[row].Guid), x, y, w, h, fltk.ALIGN_CENTER)
		}
	case fltk.ContextColHeader:
		fltk.DrawBox(fltk.UP_BOX, x, y, w, h, 0x8f8f8fff)
		fltk.Color(0x00000000)
		if column == 0 {
			fltk.Draw("Name", x, y, w, h, fltk.ALIGN_CENTER)
		} else if column == 1 {
			fltk.Draw("State", x, y, w, h, fltk.ALIGN_CENTER)
		}
	}
}

func (t *baseTab) handlePortalListEvent(event fltk.Event) bool {
	fmt.Println("portal list event", event)
	return false
}
