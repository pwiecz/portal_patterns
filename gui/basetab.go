package main

import (
	"fmt"
	"image/color"
	"os"
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
	portalColor(string) color.NRGBA
	portalLabel(string) string
	solutionString() string
	onReset()
	onContextMenu(x, y float32)
}

type baseTab struct {
	app             fyne.App
	parent          fyne.Window
	configuration   *configuration.Configuration
	tileFetcher     *osm.MapTiles
	add             *widget.Button
	reset           *widget.Button
	find            *widget.Button
	save            *widget.Button
	copy            *widget.Button
	solutionLabel   *widget.Label
	progress        *widget.ProgressBar
	solutionMap     *SolutionMap
	portals         []lib.Portal
	selectedPortals map[string]struct{}
	disabledPortals map[string]struct{}
	pattern         pattern
	name            string
}

func NewBaseTab(app fyne.App, parent fyne.Window, name string, pattern pattern, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *baseTab {
	t := &baseTab{
		app:           app,
		parent:        parent,
		name:          name,
		pattern:       pattern,
		configuration: conf,
		tileFetcher:   tileFetcher,
	}
	t.add = widget.NewButton("Add", t.onAdd)
	t.reset = widget.NewButton("Reset", t.onReset)
	t.reset.Disable()

	t.find = widget.NewButton("Search", func() { go t.pattern.search() })
	t.find.Disable()
	t.save = widget.NewButton("Save Solution", t.onSaveSolution)
	t.save.Disable()
	t.copy = widget.NewButton("Copy to Clipboard", func() {
		parent.Clipboard().SetContent(t.pattern.solutionString())
	})
	t.copy.Disable()
	t.solutionLabel = widget.NewLabel("")
	t.progress = widget.NewProgressBar()
	t.progress.Min = 0
	t.progress.Max = 1
	//	t.portalList.SetColumnWidth(0, t.portalList.
	// t.portalList = NewPortalList(parent)
	// t.portalList.OnPortalRightClick(func(guid string, x, y int) {
	// 	t.pattern.onPortalContextMenu(guid, x, y)
	// })
	// t.portalList.OnSelectionChanged(func() {
	// 	selectedPortals := t.portalList.SelectedPortals()
	// 	t.OnSelectionChanged(selectedPortals)
	// })

	t.selectedPortals = make(map[string]struct{})
	t.disabledPortals = make(map[string]struct{})
	return t
}

func (t *baseTab) onSaveSolution() {
	fileSaveDialog := dialog.NewFileSave(t.onSaveFileChosen, t.parent)
	lister, err := storage.ListerForURI(storage.NewFileURI(t.configuration.PortalsDirectory))
	if err == nil {
		fileSaveDialog.SetLocation(lister)
	}
	fileSaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))

	fileSaveDialog.Show()
}
func (t *baseTab) onSaveFileChosen(fileUri fyne.URIWriteCloser, err error) {
	if err != nil || fileUri == nil {
		return
	}
	filename := fileUri.URI().Path()
	file, err := os.Create(filename)
	if err != nil {
		dialog.ShowInformation("Error", "Cannot create file", t.parent)
		return
	}
	if _, err := file.WriteString(t.pattern.solutionString()); err != nil {
		dialog.ShowInformation("Error", "Error writing to file", t.parent)
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		dialog.ShowInformation("Error", "Error writing to file", t.parent)
	}
}
func (t *baseTab) onAdd() {
	fileOpenDialog := dialog.NewFileOpen(t.onFileChosen, t.parent)
	lister, err := storage.ListerForURI(storage.NewFileURI(t.configuration.PortalsDirectory))
	if err == nil {
		fileOpenDialog.SetLocation(lister)
	}
	fileOpenDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json", ".csv"}))

	fileOpenDialog.Show()
}

func (t *baseTab) onFileChosen(file fyne.URIReadCloser, err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
	if file == nil {
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
	t.selectedPortals = make(map[string]struct{})
	t.disabledPortals = make(map[string]struct{})
	if t.solutionMap != nil {
		t.solutionMap.Clear()
	}
	t.pattern.onReset()
	t.solutionLabel.SetText("")
	t.progress.SetValue(0)
	t.reset.Disable()
	t.find.Disable()
	t.save.Disable()
	t.copy.Disable()
}

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
		t.solutionMap.OnSelectionCleared = t.OnSelectionCleared
		t.solutionMap.OnPortalSelected = t.OnPortalSelected
		t.solutionMap.OnContextMenu = t.pattern.onContextMenu
		t.solutionMap.OnPortalContextMenu = t.onPortalContextMenu
		solutionMapWin := t.app.NewWindow(t.name + " OpenStreetMap")
		solutionMapWin.SetContent(t.solutionMap)
		solutionMapWin.SetOnClosed(func() { t.solutionMap = nil })
		solutionMapWin.Resize(fyne.NewSize(800, 600))
		/*		t.solutionMap.OnPortalLeftClick(func(guid string) {
				t.OnPortalSelected(guid)
				})
				t.solutionMap.OnPortalRightClick(func(guid string, x, y int) {
					t.pattern.onPortalContextMenu(guid, x, y)
				})
				t.solutionMap.ShowNormal()
				//tk.Update()
			}*/
		solutionMapWin.Show()
		t.solutionMap.Refresh()
	}
	t.solutionMap.SetPortals(t.portals)
	for _, portal := range t.portals {
		t.portalStateChanged(portal.Guid)
	}
}

func (t *baseTab) onPortalContextMenu(guid string, x, y float32) {
	if _, ok := t.selectedPortals[guid]; !ok {
		t.OnPortalSelected(guid, false)
	}
	t.pattern.onContextMenu(x, y)
}

func (t *baseTab) OnSelectionCleared() {
	oldSelection := t.selectedPortals
	t.selectedPortals = make(map[string]struct{})
	for guid := range oldSelection {
		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}
func (t *baseTab) OnPortalSelected(guid string, addedToSelection bool) {
	if _, ok := t.selectedPortals[guid]; ok {
		if addedToSelection || len(t.selectedPortals) == 1 {
			return
		}
	}
	if !addedToSelection {
		t.OnSelectionCleared()
	}
	t.selectedPortals[guid] = struct{}{}
	t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))

	//	t.OnSelectionChanged([]string{guid})
	//	if t.portalList != nil {
	//		t.portalList.ScrollToPortal(guid)
	//	}
	if t.solutionMap != nil {
		t.solutionMap.ScrollToPortal(guid)
	}
}

func (t *baseTab) portalStateChanged(guid string) {
	// 	if t.portalList != nil {
	// 		t.portalList.SetPortalState(guid, t.pattern.portalLabel(guid))
	// 	}
	// 	if t.solutionMap != nil {
	// 		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	// 	}
}

func (t *baseTab) enableSelectedPortals() {
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; !ok {
			continue
		}
		delete(t.disabledPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}

func (t *baseTab) getDisabledPortals() []lib.Portal {
	portals := []lib.Portal{}
	for _, portal := range t.portals {
		if _, ok := t.disabledPortals[portal.Guid]; ok {
			portals = append(portals, portal)
		}
	}
	return portals
}
