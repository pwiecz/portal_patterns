package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
	"golang.org/x/exp/maps"
)

type Portals struct {
	portals         []lib.Portal
	portalMap       map[string]lib.Portal
	selectedPortals map[string]struct{}
	disabledPortals map[string]struct{}
}

func NewPortals() *Portals {
	return &Portals{
		portalMap:       make(map[string]lib.Portal),
		selectedPortals: make(map[string]struct{}),
		disabledPortals: make(map[string]struct{}),
	}
}

type MainWindow struct {
	*fltk.Window
	configuration      *configuration.Configuration
	add, reset         *fltk.Button
	search             *fltk.Button
	export             *fltk.Button
	copy               *fltk.Button
	solutionLabel      *fltk.Box
	progress           *fltk.Progress
	lastProgressUpdate time.Time
	tabs               *fltk.Tabs
	mapWindow          *MapWindow
	portalList         *PortalList
	portals            *Portals
	homogeneous        *homogeneousTab
	herringbone        *herringboneTab
	doubleHerringbone  *doubleHerringboneTab
	cobweb             *cobwebTab
	flipField          *flipFieldTab
	droneFlight        *droneFlightTab
	threeCorners       *threeCornersTab
	selectedTab        int
	searchInProgress   bool
}

func NewMainWindow(conf *configuration.Configuration) *MainWindow {
	w := &MainWindow{}
	w.Window = fltk.NewWindow(1600, 900)
	w.configuration = conf
	w.portals = NewPortals()
	w.SetCallback(w.onClose)
	w.Begin()
	mainPack := fltk.NewPack(0, 0, 1600, 900)
	mainPack.SetType(fltk.VERTICAL)
	menuBar := fltk.NewMenuBar(0, 0, 1600, 30)
	menuBar.AddEx("&File/&Load", fltk.CTRL+int('o'), w.onLoadPressed, 0)
	menuBar.AddEx("&File/&Save", fltk.CTRL+int('s'), w.onSavePressed, 0)
	menuBar.AddEx("&Select/Select &All", fltk.CTRL+int('a'), w.onSelectAll, 0)
	menuBar.AddEx("&Select/&Invert", fltk.CTRL+int('i'), w.onInvertSelection, 0)
	menuBar.AddEx("&Select/&Rectangular Selection", fltk.ALT+int('r'), w.onRectangularSelection, 0)
	menuBar.AddEx("&View/Zoom &In", fltk.CTRL+int('+'), w.onZoomIn, 0)
	menuBar.AddEx("&View/Zoom &Out", fltk.CTRL+int('-'), w.onZoomOut, 0)
	pack := fltk.NewPack(0, 0, 1600, 870)
	pack.SetType(fltk.HORIZONTAL)
	tileFetcher := osm.NewMapTiles()
	w.mapWindow = NewMapWindow(0, 0, 900, 870, "", tileFetcher, w.Window)
	w.mapWindow.SetSelectionChangeCallback(w.OnSelectionChanged)
	w.mapWindow.SetAddedToSelectionCallback(func(selection map[string]struct{}) {
		selectionCopy := make(map[string]struct{})
		for guid := range w.portals.selectedPortals {
			selectionCopy[guid] = struct{}{}
		}
		for guid := range selection {
			selectionCopy[guid] = struct{}{}
		}
		w.OnSelectionChanged(selectionCopy)
	})
	w.mapWindow.SetRightClickCallback(func(guid string, x, y int) {
		if guid != "" {
			if _, ok := w.portals.selectedPortals[guid]; !ok {
				selection := make(map[string]struct{})
				selection[guid] = struct{}{}
				w.OnSelectionChanged(selection)
			}
		}
		w.onContextMenu(x, y)
	})
	rightPack := fltk.NewPack(0, 0, 700, 870)
	rightPack.SetType(fltk.VERTICAL)
	topButtonPack := fltk.NewPack(0, 0, 700, 30)
	topButtonPack.SetType(fltk.HORIZONTAL)
	topButtonPack.SetSpacing(5)

	w.add = fltk.NewButton(0, 0, 101, 30, "Add portals")
	w.add.SetCallback(w.onAddPortalsPressed)
	w.reset = fltk.NewButton(0, 0, 113, 30, "Reset portals")
	w.reset.Deactivate()
	w.reset.SetCallback(w.onResetPortalsPressed)
	topButtonPack.End()

	w.tabs = fltk.NewTabs(0, 0, 700, 200)
	w.tabs.SetCallbackCondition(fltk.WhenChanged)
	w.homogeneous = newHomogeneousTab(w.portals)
	w.herringbone = newHerringboneTab(w.portals)
	w.doubleHerringbone = newDoubleHerringboneTab(w.portals)
	w.cobweb = newCobwebTab(w.portals)
	w.droneFlight = newDroneFlightTab(w.portals)
	w.flipField = newFlipFieldTab(w.portals)
	w.threeCorners = newThreeCornersTab(w.portals)
	w.tabs.Add(w.homogeneous)
	w.tabs.Add(w.herringbone)
	w.tabs.Add(w.doubleHerringbone)
	w.tabs.Add(w.cobweb)
	w.tabs.Add(w.droneFlight)
	w.tabs.Add(w.flipField)
	w.tabs.Add(w.threeCorners)
	w.tabs.SetCallback(func() { w.onTabSelected(w.tabs.Value()) })
	w.tabs.End()
	// Mark one random tab as resizable, as per www.fltk.org/doc-1.3/classFl__Tabs.html - "resizing caveats"
	w.tabs.Resizable(w.flipField)

	searchSaveCopyPack := fltk.NewPack(0, 0, 700, 30)
	searchSaveCopyPack.SetType(fltk.HORIZONTAL)
	searchSaveCopyPack.SetSpacing(5)
	w.search = fltk.NewButton(0, 0, 70, 30, "Search")
	w.search.Deactivate()
	w.search.SetCallback(w.onSearchPressed)
	w.export = fltk.NewButton(0, 0, 147, 30, "Export Draw Tools")
	w.export.Deactivate()
	w.export.SetCallback(w.onExportPressed)
	w.copy = fltk.NewButton(0, 0, 140, 30, "Copy Draw Tools")
	w.copy.Deactivate()
	w.copy.SetCallback(w.onCopyPressed)
	searchSaveCopyPack.End()

	w.solutionLabel = fltk.NewBox(fltk.NO_BOX, 0, 0, 700, 30)
	w.solutionLabel.SetAlign(fltk.ALIGN_INSIDE | fltk.ALIGN_LEFT)

	w.progress = fltk.NewProgress(0, 0, 700, 30)
	w.progress.SetSelectionColor(0x4444ff00)
	w.lastProgressUpdate = time.Now()

	portalListPack := fltk.NewPack(0, 0, 700, 590)
	portalListPack.SetType(fltk.HORIZONTAL)
	w.portalList = NewPortalList(0, 0, 700, 590)
	w.portalList.SetSelectionChangeCallback(func() { w.OnSelectionChanged(w.portalList.selectedPortals) })
	w.portalList.SetContextMenuCallback(w.onContextMenu)
	// Add an empty group that will resize horizontally instead of the portal list
	// when the main window grows horizontally.
	dummyGroup := fltk.NewGroup(0, 0, 0, 590)
	portalListPack.End()
	portalListPack.Resizable(dummyGroup)
	rightPack.End()
	rightPack.Resizable(portalListPack)
	pack.End()
	pack.Resizable(w.mapWindow)
	mainPack.End()
	mainPack.Resizable(pack)
	w.End()
	w.Resizable(mainPack)
	return w
}

func (w *MainWindow) onClose() {
	if fltk.EventType() == fltk.SHORTCUT && fltk.EventKey() == fltk.ESCAPE {
		// Don't close the main window when user just presses Escape.
		return
	}
	w.Hide()
}

func (w *MainWindow) selectedPattern() pattern {
	switch w.selectedTab {
	case 0:
		return w.homogeneous
	case 1:
		return w.herringbone
	case 2:
		return w.doubleHerringbone
	case 3:
		return w.cobweb
	case 4:
		return w.droneFlight
	case 5:
		return w.flipField
	case 6:
		return w.threeCorners
	}
	return nil
}

func (w *MainWindow) onTabSelected(selectedIx int) {
	w.selectedTab = selectedIx
	selectedPattern := w.selectedPattern()
	for guid := range w.portals.portalMap {
		fill, stroke := selectedPattern.portalColor(guid)
		w.mapWindow.SetPortalColor(guid, fill, stroke)
		w.portalList.SetPortalLabel(guid, selectedPattern.portalLabel(guid))
	}
	if selectedPattern.hasSolution() {
		w.solutionLabel.SetLabel(selectedPattern.solutionInfoString())
		w.mapWindow.SetPaths(selectedPattern.solutionPaths())
		w.export.Activate()
		w.copy.Activate()
	} else {
		w.solutionLabel.SetLabel("")
		w.mapWindow.SetPaths(nil)
		w.export.Deactivate()
		w.copy.Deactivate()
	}
	if !w.searchInProgress {
		if selectedPattern.finishedSearching() {
			w.progress.SetValue(w.progress.Maximum())
		} else {
			w.progress.SetValue(0)
		}
	}
	w.mapWindow.Redraw()
	w.portalList.Redraw()
}

func (w *MainWindow) onRectangularSelection() {
	w.mapWindow.SetSelectionMode(RectangularSelection)
	w.mapWindow.Redraw()
}
func (w *MainWindow) onSelectAll() {
	selection := make(map[string]struct{})
	for _, portal := range w.portals.portals {
		selection[portal.Guid] = struct{}{}
	}
	w.OnSelectionChanged(selection)
}
func (w *MainWindow) onInvertSelection() {
	selection := make(map[string]struct{})
	for _, portal := range w.portals.portals {
		if _, ok := w.portals.selectedPortals[portal.Guid]; !ok {
			selection[portal.Guid] = struct{}{}
		}
	}
	w.OnSelectionChanged(selection)
}

func (w *MainWindow) OnSelectionChanged(selectedPortals map[string]struct{}) {
	if maps.Equal(selectedPortals, w.portals.selectedPortals) {
		return
	}
	var selectedBefore map[string]struct{}
	w.portals.selectedPortals, selectedBefore = maps.Clone(selectedPortals), w.portals.selectedPortals
	w.portalList.SetSelectedPortals(w.portals.selectedPortals)
	selectedPattern := w.selectedPattern()
	for guid := range selectedBefore {
		fill, stroke := selectedPattern.portalColor(guid)
		w.mapWindow.SetPortalColor(guid, fill, stroke)
	}
	selectedGUID := ""
	for guid := range w.portals.selectedPortals {
		selectedGUID = guid
		fill, stroke := selectedPattern.portalColor(guid)
		w.mapWindow.SetPortalColor(guid, fill, stroke)
		w.mapWindow.Raise(guid)
	}
	if len(w.portals.selectedPortals) == 1 {
		w.mapWindow.ScrollToPortal(selectedGUID)
		w.portalList.ScrollToPortal(selectedGUID)
	}
	w.mapWindow.Redraw()
}

func (w *MainWindow) onContextMenu(x, y int) {
	selectedPattern := w.selectedPattern()
	menu := selectedPattern.contextMenu()
	if menu == nil || len(menu.items) == 0 {
		return
	}
	mb := fltk.NewMenuButton(x, y, 100, 100, menu.header)
	mb.SetType(fltk.POPUP3)
	for _, item := range menu.items {
		mb.Add(item.label, item.callback)
	}
	mb.Popup()
	mb.Destroy()
	for guid := range w.portals.portalMap {
		fill, stroke := selectedPattern.portalColor(guid)
		w.mapWindow.SetPortalColor(guid, fill, stroke)
		w.portalList.SetPortalLabel(guid, selectedPattern.portalLabel(guid))
	}
	w.portalList.Redraw()
	w.mapWindow.Redraw()
}
func (w *MainWindow) onZoomIn() {
	w.mapWindow.ZoomIn()
	w.mapWindow.Redraw()
}
func (w *MainWindow) onZoomOut() {
	w.mapWindow.ZoomOut()
	w.mapWindow.Redraw()
}
func (w *MainWindow) onLoadPressed() {
	fileChooser := fltk.NewFileChooser(w.configuration.PortalsDirectory, "PP files (*.pp)", fltk.FileChooser_SINGLE, "Select project file")
	fileChooser.SetPreview(false)
	defer fileChooser.Destroy()
	fileChooser.Popup()
	selectedFilenames := fileChooser.Selection()
	if len(selectedFilenames) != 1 {
		return
	}
	filename := selectedFilenames[0]
	file, err := os.Open(filename)
	if err != nil {
		fltk.MessageBox("Error loading", "Couldn't open file "+filename+"\n"+err.Error())
		return
	}
	defer file.Close()
	if err := w.decode(file); err != nil {
		fltk.MessageBox("Error loading", "Error while loading "+filename+"\n"+err.Error())
		w.onResetPortalsPressed()
		return
	}
	w.SetLabel(filepath.Base(filename))
}
func (w *MainWindow) onSavePressed() {
	fileChooser := fltk.NewFileChooser(w.configuration.PortalsDirectory, "PP files (*.pp)", fltk.FileChooser_CREATE, "Select project file")
	fileChooser.SetPreview(false)
	defer fileChooser.Destroy()
	fileChooser.Popup()
	selectedFilenames := fileChooser.Selection()
	if len(selectedFilenames) != 1 {
		return
	}
	filename := selectedFilenames[0]
	if filepath.Ext(filename) != ".pp" {
		filename += ".pp"
	}
	if stat, err := os.Stat(filename); err == nil {
		if stat.IsDir() {
			fltk.MessageBox("Directory selected", "Selected file "+filename+" is a directory")
			return
		}
		answer := fltk.ChoiceDialog("File already exists.\nDo you want to overwrite it?", "Yes", "No")
		if answer != 0 {
			return
		}
	}
	file, err := os.Create(filename)
	if err != nil {
		fltk.MessageBox("Error saving", "Couldn't create file "+filename+"\n"+err.Error())
		return
	}
	defer file.Close()
	if err := w.encode(file); err != nil {
		fltk.MessageBox("Error saving", "Error while saving to "+filename+"\n"+err.Error())
		return
	}
	w.SetLabel(filepath.Base(filename))
}
func (w *MainWindow) onAddPortalsPressed() {
	fileChooser := fltk.NewFileChooser(w.configuration.PortalsDirectory, "JSON files (*.json)\tCSV files (*.csv)", fltk.FileChooser_MULTI, "Select portals file")
	fileChooser.SetPreview(false)
	defer fileChooser.Destroy()
	fileChooser.Popup()
	selectedFilenames := fileChooser.Selection()
	if len(selectedFilenames) == 0 {
		return
	}
	for _, filename := range selectedFilenames {
		w.onPortalsFileSelected(filename)
	}
}

func (w *MainWindow) onPortalsFileSelected(filename string) {
	portalsDir, _ := filepath.Split(filename)
	w.configuration.PortalsDirectory = portalsDir
	configuration.SaveConfiguration(w.configuration)
	portals, err := lib.ParseFile(filename)
	if err != nil {
		fltk.MessageBox("Error loading", "Couldn't read portals from file "+filename+"\n"+err.Error())
		return
	}
	w.addPortals(portals)
	w.onPortalsChanged()
}

func (w *MainWindow) onPortalsChanged() {
	w.portalList.SetPortals(w.portals.portals)
	w.mapWindow.SetPortals(w.portals.portals)
	selectedPattern := w.selectedPattern()
	for guid := range w.portals.portalMap {
		fill, stroke := selectedPattern.portalColor(guid)
		w.mapWindow.SetPortalColor(guid, fill, stroke)
		w.portalList.SetPortalLabel(guid, selectedPattern.portalLabel(guid))
	}
	if len(w.portals.portals) >= 3 {
		w.search.Activate()
	} else {
		w.search.Deactivate()
	}
	if len(w.portals.portals) > 0 {
		w.reset.Activate()
	}
	w.mapWindow.Redraw()
	w.portalList.Redraw()
}

func (w *MainWindow) addPortals(portals []lib.Portal) {
	portalMap := maps.Clone(w.portals.portalMap)
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
	w.portals.portals = append(w.portals.portals, newPortals...)
	w.portals.portalMap = portalMap
}

func (w *MainWindow) onResetPortalsPressed() {
	w.portals.portals = w.portals.portals[:0]
	w.portals.portalMap = make(map[string]lib.Portal)
	w.portals.selectedPortals = make(map[string]struct{})
	w.portals.disabledPortals = make(map[string]struct{})
	w.progress.SetValue(0)
	w.reset.Deactivate()
	w.search.Deactivate()
	w.export.Deactivate()
	w.copy.Deactivate()
	w.mapWindow.SetPortals(w.portals.portals)
	w.mapWindow.SetPaths(nil)
	w.portalList.SetPortals(w.portals.portals)
	w.solutionLabel.SetLabel("")
	w.homogeneous.onReset()
	w.herringbone.onReset()
	w.doubleHerringbone.onReset()
	w.cobweb.onReset()
	w.droneFlight.onReset()
	w.flipField.onReset()
	w.threeCorners.onReset()
	w.SetLabel("")
	w.mapWindow.Redraw()
}

func (w *MainWindow) onSearchPressed() {
	w.searchInProgress = true
	w.add.Deactivate()
	w.reset.Deactivate()
	w.search.Deactivate()
	w.export.Deactivate()
	w.copy.Deactivate()
	w.portalList.Deactivate()
	w.progress.SetValue(0)
	selectedPattern := w.selectedPattern()
	selectedPattern.onSearch(w.progressCallback, w.onSearchDone)
}
func (w *MainWindow) progressCallback(val, max int) {
	now := time.Now()
	sinceLastUpdate := now.Sub(w.lastProgressUpdate)
	if sinceLastUpdate.Milliseconds() < 10 {
		return
	}
	w.lastProgressUpdate = now
	fltk.Awake(func() {
		if maxF := float64(max); maxF != w.progress.Maximum() {
			w.progress.SetMaximum(maxF)
		}
		w.progress.SetValue(float64(val))
	})
}
func (w *MainWindow) onSearchDone() {
	w.searchInProgress = false
	w.progress.SetValue(w.progress.Maximum())
	w.add.Activate()
	w.reset.Activate()
	w.search.Activate()
	w.portalList.Activate()
	selectedPattern := w.selectedPattern()
	if selectedPattern.hasSolution() {
		w.export.Activate()
		w.copy.Activate()
		w.solutionLabel.SetLabel(selectedPattern.solutionInfoString())
		w.mapWindow.SetPaths(selectedPattern.solutionPaths())
	} else {
		w.solutionLabel.SetLabel("No solution found")
		w.mapWindow.SetPaths(nil)
	}
	w.mapWindow.Redraw()
}
func (w *MainWindow) onExportPressed() {
	fileChooser := fltk.NewFileChooser(w.configuration.PortalsDirectory, "JSON files (*.json)", fltk.FileChooser_CREATE, "Select draw tools file")
	fileChooser.SetPreview(false)
	defer fileChooser.Destroy()
	fileChooser.Popup()
	selectedFilenames := fileChooser.Selection()
	if len(selectedFilenames) != 1 {
		return
	}
	filename := selectedFilenames[0]
	w.onDrawToolsFileSelected(filename)
}
func (w *MainWindow) onDrawToolsFileSelected(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fltk.MessageBox("Error exporting", "Couldn't create file "+filename+"\n"+err.Error())
		return
	}
	defer file.Close()
	if _, err := file.WriteString(w.selectedPattern().solutionDrawToolsString()); err != nil {
		fltk.MessageBox("Error exporting", "Error writing to file "+filename+"\n"+err.Error())
	}
}
func (w *MainWindow) onCopyPressed() {
	fltk.CopyToClipboard(w.selectedPattern().solutionDrawToolsString())
}

type state struct {
	Portals           []lib.Portal           `json:"portals"`
	DisabledPortals   []string               `json:"disabledPortals"`
	SelectedPortals   []string               `json:"selectedPortals"`
	SelectedTab       int                    `json:"selectedTab"`
	Homogeneous       homogeneousState       `json:"homogeneous"`
	Herringbone       herringboneState       `json:"herringbone"`
	DoubleHerringbone doubleHerringboneState `json:"doubleHerringbone"`
	Cobweb            cobwebState            `json:"cobweb"`
	DroneFlight       droneFlightState       `json:"droneFlight"`
	FlipField         flipFieldState         `json:"flipField"`
	ThreeCorners      threeCornersState      `json:"threeCorners"`
}

func (w *MainWindow) encode(writer io.Writer) error {
	s := state{
		Portals:           w.portals.portals,
		SelectedTab:       w.selectedTab,
		Homogeneous:       w.homogeneous.state(),
		Herringbone:       w.herringbone.state(),
		DoubleHerringbone: w.doubleHerringbone.state(),
		Cobweb:            w.cobweb.state(),
		DroneFlight:       w.droneFlight.state(),
		FlipField:         w.flipField.state(),
		ThreeCorners:      w.threeCorners.state(),
	}

	s.DisabledPortals = maps.Keys(w.portals.disabledPortals)
	s.SelectedPortals = maps.Keys(w.portals.selectedPortals)
	return json.NewEncoder(writer).Encode(s)
}

func (w *MainWindow) decode(reader io.Reader) error {
	s := state{}
	err := json.NewDecoder(reader).Decode(&s)
	if err != nil {
		return err
	}
	w.portals.portals = s.Portals
	w.portals.portalMap = make(map[string]lib.Portal)
	for _, portal := range w.portals.portals {
		w.portals.portalMap[portal.Guid] = portal
	}
	w.portals.disabledPortals = make(map[string]struct{})
	for _, disabledGUID := range s.DisabledPortals {
		if _, ok := w.portals.portalMap[disabledGUID]; !ok {
			return fmt.Errorf("invalid disabled portal %s", disabledGUID)
		}
		w.portals.disabledPortals[disabledGUID] = struct{}{}
	}
	w.portals.selectedPortals = make(map[string]struct{})
	for _, selectedGUID := range s.SelectedPortals {
		if _, ok := w.portals.portalMap[selectedGUID]; !ok {
			return fmt.Errorf("invalid selected portal %s", selectedGUID)
		}
		w.portals.selectedPortals[selectedGUID] = struct{}{}
	}
	if err := w.homogeneous.load(s.Homogeneous); err != nil {
		return err
	}
	if err := w.herringbone.load(s.Herringbone); err != nil {
		return err
	}
	if err := w.doubleHerringbone.load(s.DoubleHerringbone); err != nil {
		return err
	}
	if err := w.cobweb.load(s.Cobweb); err != nil {
		return err
	}
	if err := w.droneFlight.load(s.DroneFlight); err != nil {
		return err
	}
	if err := w.flipField.load(s.FlipField); err != nil {
		return err
	}
	if err := w.threeCorners.load(s.ThreeCorners); err != nil {
		return err
	}
	if s.SelectedTab < 0 || s.SelectedTab > 6 {
		return fmt.Errorf("invalid selected tab %d", s.SelectedTab)
	}
	w.selectedTab = s.SelectedTab
	w.tabs.SetValue(w.selectedTab)
	w.onPortalsChanged()
	w.onTabSelected(w.selectedTab)

	return nil
}

func main() {
	runtime.LockOSThread()
	conf := configuration.LoadConfiguration()
	// Disable screen scaling, as we don't handle it well.
	for i := 0; i < fltk.ScreenCount(); i++ {
		fltk.SetScreenScale(i, 1.0)
	}
	fltk.SetKeyboardScreenScaling(false)
	w := NewMainWindow(conf)
	fltk.Lock()
	w.Show()

	fltk.Run()

	w.mapWindow.Destroy()
	w.Destroy()
}
