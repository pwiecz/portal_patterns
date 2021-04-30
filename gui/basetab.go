package main

import (
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/golang/geo/s2"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type pattern interface {
	search()
	portalColor(string) string
	portalLabel(string) string
	solutionString() string
	onReset()
	onPortalContextMenu(guid string, x, y int)
}

type baseTab struct {
	app           fyne.App
	parent        fyne.Window
	configuration *configuration.Configuration
	tileFetcher   *osm.MapTiles
	add           *widget.Button
	reset         *widget.Button
	find          *widget.Button
	save          *widget.Button
	copy          *widget.Button
	solutionLabel *widget.Label
	progress      *widget.ProgressBar
	//	portalList      *PortalList
	//	portalScrollBar *tk.ScrollBar
	solutionMap     *SolutionMap
	portals         []lib.Portal
	selectedPortals map[string]bool
	disabledPortals map[string]bool
	pattern         pattern
	name            string
}

func NewBaseTab(app fyne.App, parent fyne.Window, name string, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *baseTab {
	t := &baseTab{
		app:           app,
		parent:        parent,
		name:          name,
		configuration: conf,
		tileFetcher:   tileFetcher,
	}
	t.add = widget.NewButton("Add", func() { t.onAdd() })
	t.reset = widget.NewButton("Reset", func() { t.onReset() })

	t.find = widget.NewButton("Search", func() { go t.pattern.search() })
	t.find.Disable()
	t.save = widget.NewButton("Save Solution", func() {})
	// t.save.OnCommand(func() {
	// 	filename, err := tk.GetSaveFile(parent, "Select file for solution", true, ".json",
	// 		[]tk.FileType{{Info: "JSON file", Ext: ".json"}}, conf.PortalsDirectory, "")
	// 	if err != nil || filename == "" {
	// 		return
	// 	}
	// 	file, err := os.Create(filename)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	defer file.Close()
	// 	file.WriteString(t.pattern.solutionString())
	// })
	t.save.Disable()
	t.copy = widget.NewButton("Copy to Clipboard", func() {})
	// t.copy.OnCommand(func() {
	// 	tk.ClearClipboard()
	// 	tk.AppendToClipboard(t.pattern.solutionString())
	// })
	t.copy.Disable()
	t.solutionLabel = widget.NewLabel("")
	t.progress = widget.NewProgressBar()
	t.progress.Min = 0
	t.progress.Max = 1
	// t.portalList = NewPortalList(parent)
	// t.portalList.OnPortalRightClick(func(guid string, x, y int) {
	// 	t.pattern.onPortalContextMenu(guid, x, y)
	// })
	// t.portalList.OnSelectionChanged(func() {
	// 	selectedPortals := t.portalList.SelectedPortals()
	// 	t.OnSelectionChanged(selectedPortals)
	// })

	t.selectedPortals = make(map[string]bool)
	t.disabledPortals = make(map[string]bool)
	return t
}

func (t *baseTab) onAdd() {
	fileOpenDialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) { t.onFileChosen(file, err) }, t.parent)
	lister, err := storage.ListerForURI(storage.NewFileURI(t.configuration.PortalsDirectory))
	if err == nil {
		fileOpenDialog.SetLocation(lister)
	}
	fileOpenDialog.Show()
}

func (t *baseTab) onFileChosen(file fyne.URIReadCloser, err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
	filename := file.URI().Path()
	portalsDir, _ := filepath.Split(filename)
	t.configuration.PortalsDirectory = portalsDir
	configuration.SaveConfiguration(t.configuration)
	portals, err := lib.ParseFile(filename)
	if err != nil {
		dialog.ShowInformation("Could not read file", fmt.Sprintf("Error reading file:\n%v", err), t.parent)
		return
	}
	t.addPortals(portals)
}

func (t *baseTab) onReset() {
	t.portals = nil
	t.selectedPortals = make(map[string]bool)
	t.disabledPortals = make(map[string]bool)
	if t.solutionMap != nil {
		t.solutionMap.Clear()
	}
}

// func (t *baseTab) onAdd() {
// 	filenames, err := tk.GetOpenMultipleFile(t, "Choose portals file",
// 		[]tk.FileType{
// 			{Info: "JSON file", Ext: ".json"},
// 			{Info: "CSV file", Ext: ".csv"},
// 		}, t.configuration.PortalsDirectory, "")
// 	if err != nil || len(filenames) == 0 {
// 		return
// 	}
// 	portalsDir, _ := filepath.Split(filenames[0])
// 	t.configuration.PortalsDirectory = portalsDir
// 	configuration.SaveConfiguration(t.configuration)
// 	for _, filename := range filenames {
// 		portals, err := lib.ParseFile(filename)
// 		if err != nil {
// 			tk.MessageBox(t, "Could not read file", fmt.Sprintf("Error reading file:\n%v", err),
// 				"", "", tk.MessageBoxIconError, tk.MessageBoxTypeOk)
// 			return
// 		}
// 		t.addPortals(portals)
// 	}
// }

// func (t *baseTab) onReset() {
// 	t.portals = ([]lib.Portal)(nil)
// 	t.selectedPortals = make(map[string]bool)
// 	t.disabledPortals = make(map[string]bool)
// 	t.reset.SetState(tk.StateDisable)
// 	t.find.SetState(tk.StateDisable)
// 	t.save.SetState(tk.StateDisable)
// 	if t.solutionMap != nil {
// 		t.solutionMap.Clear()
// 	}
// 	if t.portalList != nil {
// 		t.portalList.Clear()
// 	}
// 	t.solutionLabel.SetText("")
// }

func (t *baseTab) onProgress(val int, max int) {
	value := float64(val) / float64(max)
	t.progress.SetValue(value)
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
					dialog.ShowInformation("Conflicting portals", "Portal with guid \""+portal.Guid+"\" already loaded with different location:\n"+
						portal.Name+"\n"+portal.LatLng.String()+" vs "+existing.LatLng.String(), t.parent)
					return
				}
				dialog.ShowInformation("Conflicting portals", "Portal with guid \""+portal.Guid+"\" already loaded with different name and location:\n"+
					portal.Name+" vs "+existing.Name+"\n"+portal.LatLng.String()+" vs "+existing.LatLng.String(), t.parent)
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
		dialog.ShowInformation("Too distant portals", "Distances between portals are too large", t.parent)
		return
	}
	t.portals = append(t.portals, newPortals...)
	if len(t.portals) > 0 {
		t.reset.Enable()
	}
	if len(t.portals) >= 3 {
		t.find.Enable()
	}
	if t.solutionMap == nil {
		t.solutionMap = NewSolutionMap(t.tileFetcher)
		solutionMapWin := t.app.NewWindow(t.name + " OpenStreetMap")
		solutionMapWin.SetContent(t.solutionMap)
		solutionMapWin.SetOnClosed(func() { t.solutionMap = nil })
		solutionMapWin.Resize(fyne.Size{800, 600})
		/*		t.solutionMap.OnPortalLeftClick(func(guid string) {
				t.OnPortalSelected(guid)
				})
				t.solutionMap.OnPortalRightClick(func(guid string, x, y int) {
					t.pattern.onPortalContextMenu(guid, x, y)
				})
				t.solutionMap.ShowNormal()
				//tk.Update()
			}*/
		//t.portalList.SetPortals(t.portals)
		solutionMapWin.Show()
		t.solutionMap.Refresh()
	}
	t.solutionMap.SetPortals(t.portals)
	for _, portal := range t.portals {
		t.portalStateChanged(portal.Guid)
	}
}

// func stringMapsAreTheSame(map1 map[string]bool, map2 map[string]bool) bool {
// 	for s := range map1 {
// 		if !map2[s] {
// 			return false
// 		}
// 	}
// 	for s := range map2 {
// 		if !map1[s] {
// 			return false
// 		}
// 	}
// 	return true
// }

// func (t *baseTab) OnSelectionChanged(selection []string) {
// 	selectionMap := make(map[string]bool)
// 	for _, guid := range selection {
// 		selectionMap[guid] = true
// 	}
// 	if stringMapsAreTheSame(selectionMap, t.selectedPortals) {
// 		return
// 	}
// 	oldSelected := t.selectedPortals
// 	t.selectedPortals = selectionMap
// 	if t.portalList != nil {
// 		t.portalList.SetSelectedPortals(t.selectedPortals)
// 	}
// 	if t.solutionMap != nil {
// 		for guid := range oldSelected {
// 			t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
// 		}
// 		for _, guid := range selection {
// 			t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
// 			t.solutionMap.RaisePortal(guid)
// 		}
// 	}
// 	if len(selection) == 1 {
// 		if t.portalList != nil {
// 			t.portalList.ScrollToPortal(selection[0])
// 		}
// 		if t.solutionMap != nil {
// 			t.solutionMap.ScrollToPortal(selection[0])
// 		}
// 	}
// }
// func (t *baseTab) OnPortalSelected(guid string) {
// 	t.OnSelectionChanged([]string{guid})
// 	if t.portalList != nil {
// 		t.portalList.ScrollToPortal(guid)
// 	}
// 	if t.solutionMap != nil {
// 		t.solutionMap.ScrollToPortal(guid)
// 	}
// }

func (t *baseTab) portalStateChanged(guid string) {
	// 	if t.portalList != nil {
	// 		t.portalList.SetPortalState(guid, t.pattern.portalLabel(guid))
	// 	}
	// 	if t.solutionMap != nil {
	// 		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	// 	}
}
