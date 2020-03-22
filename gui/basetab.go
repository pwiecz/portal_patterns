package main

import "fmt"
import "os"
import "path"

import "github.com/golang/geo/s2"
import "github.com/pwiecz/atk/tk"
import "github.com/pwiecz/portal_patterns/lib"

type pattern interface {
	search()
	portalColor(string) string
	portalLabel(string) string
	solutionString() string
	onReset()
	onPortalContextMenu(guid string, x, y int)
}

type baseTab struct {
	*tk.PackLayout
	configuration   *Configuration
	add             *tk.Button
	reset           *tk.Button
	find            *tk.Button
	save            *tk.Button
	copy            *tk.Button
	solutionLabel   *tk.Label
	progress        *tk.ProgressBar
	portalList      *PortalList
	portalScrollBar *tk.ScrollBar
	solutionMap     *SolutionMap
	portals         map[string]lib.Portal
	selectedPortals map[string]bool
	disabledPortals map[string]bool
	pattern         pattern
}

func NewBaseTab(parent tk.Widget, conf *Configuration) *baseTab {
	t := &baseTab{}
	t.configuration = conf
	t.PackLayout = tk.NewVPackLayout(parent)
	t.add = tk.NewButton(parent, "Add Portals")
	t.add.OnCommand(func() {
		t.onAdd()
	})
	t.reset = tk.NewButton(parent, "Reset Portals")
	t.reset.OnCommand(func() {
		t.onReset()
	})
	t.reset.SetState(tk.StateDisable)
	t.find = tk.NewButton(parent, "Search")
	t.find.OnCommand(func() {
		t.pattern.search()
	})
	t.find.SetState(tk.StateDisable)
	t.save = tk.NewButton(parent, "Save Solution")
	t.save.OnCommand(func() {
		filename, err := tk.GetSaveFile(parent, "Select file for solution", true, ".json",
			[]tk.FileType{tk.FileType{Info: "JSON file", Ext: ".json"}}, conf.PortalsDirectory, "")
		if err != nil || filename == "" {
			return
		}
		file, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		file.WriteString(t.pattern.solutionString())
	})
	t.save.SetState(tk.StateDisable)
	t.copy = tk.NewButton(parent, "Copy to Clipboard")
	t.copy.OnCommand(func() {
		tk.ClearClipboard()
		tk.AppendToClipboard(t.pattern.solutionString())
	})
	t.copy.SetState(tk.StateDisable)
	t.solutionLabel = tk.NewLabel(parent, "")
	t.progress = tk.NewProgressBar(parent, tk.Horizontal, tk.ProgressBarAttrMaximum(1000))
	t.progress.SetDeterminateMode(true)
	t.portalList = NewPortalList(parent)
	t.portalList.OnPortalRightClick(func(guid string, x, y int) {
		t.pattern.onPortalContextMenu(guid, x, y)
	})
	t.portalList.OnSelectionChanged(func() {
		selectedPortals := t.portalList.SelectedPortals()
		t.OnSelectionChanged(selectedPortals)
	})

	t.portals = make(map[string]lib.Portal)
	t.selectedPortals = make(map[string]bool)
	t.disabledPortals = make(map[string]bool)
	return t
}

func (t *baseTab) onAdd() {
	filename, err := tk.GetOpenFile(t, "Choose portals file",
		[]tk.FileType{
			tk.FileType{Info: "JSON file", Ext: ".json"},
			tk.FileType{Info: "CSV file", Ext: ".csv"},
		}, t.configuration.PortalsDirectory, "")
	if err != nil || filename == "" {
		return
	}
	portalsDir, _ := path.Split(filename)
	t.configuration.PortalsDirectory = portalsDir
	SaveConfiguration(t.configuration)
	portals, err := lib.ParseFile(filename)
	if err != nil {
		tk.MessageBox(t, "Could not read file", fmt.Sprintf("Error reading file:\n%v", err),
			"", "", tk.MessageBoxIconError, tk.MessageBoxTypeOk)
		return
	}
	t.addPortals(portals)
}

func (t *baseTab) onReset() {
	t.portals = make(map[string]lib.Portal)
	t.selectedPortals = make(map[string]bool)
	t.disabledPortals = make(map[string]bool)
	t.reset.SetState(tk.StateDisable)
	t.find.SetState(tk.StateDisable)
	t.save.SetState(tk.StateDisable)
	if t.solutionMap != nil {
		t.solutionMap.Clear()
	}
	if t.portalList != nil {
		t.portalList.Clear()
	}
	t.solutionLabel.SetText("")
}

func (t *baseTab) onProgress(val int, max int) {
	value := float64(val) * 1000. / float64(max)
	t.progress.SetValue(value)
	tk.Update()
}

func (t *baseTab) addPortals(portals []lib.Portal) {
	newPortals := make(map[string]lib.Portal)
	for guid, portal := range t.portals {
		newPortals[guid] = portal
	}
	for _, portal := range portals {
		if existing, ok := t.portals[portal.Guid]; ok {
			if existing.LatLng.Lat != portal.LatLng.Lat ||
				existing.LatLng.Lng != portal.LatLng.Lng {
				if existing.Name == portal.Name {
					tk.MessageBox(t, "Conflicting portals", "Portal with guid \""+portal.Guid+"\" already loaded with different location",
						portal.Name+"\n"+portal.LatLng.String()+" vs "+existing.LatLng.String(), "", tk.MessageBoxIconWarning, tk.MessageBoxTypeOk)
					return
				}
				tk.MessageBox(t, "Conflicting portals", "Portal with guid \""+portal.Guid+"\" already loaded with different name and location",
					portal.Name+" vs "+existing.Name+"\n"+portal.LatLng.String()+" vs "+existing.LatLng.String(), "", tk.MessageBoxIconWarning, tk.MessageBoxTypeOk)
				return
			}
		}
		newPortals[portal.Guid] = portal
	}

	for guid0, p0 := range newPortals {
		for guid1, p1 := range newPortals {
			if guid0 == guid1 {
				continue
			}
			if s2.PointFromLatLng(p0.LatLng).Distance(s2.PointFromLatLng(p1.LatLng)) >= 1. {
				tk.MessageBox(t, "Too distant portals", "Distances between portals are too large", "E.g. "+p0.Name+" and "+p1.Name, "", tk.MessageBoxIconWarning, tk.MessageBoxTypeOk)

				return
			}
		}

	}
	t.portals = newPortals
	if len(t.portals) > 0 {
		t.reset.SetState(tk.StateNormal)
	}
	if len(t.portals) >= 3 {
		t.find.SetState(tk.StateNormal)
	}
	if t.solutionMap == nil {
		t.solutionMap = NewSolutionMap(t, "Herringbone")
		t.solutionMap.OnClose(func() bool {
			t.solutionMap = nil
			return true
		})
		t.solutionMap.OnPortalLeftClick(func(guid string) {
			t.OnPortalSelected(guid)
		})
		t.solutionMap.OnPortalRightClick(func(guid string, x, y int) {
			t.pattern.onPortalContextMenu(guid, x, y)
		})
		t.solutionMap.ShowNormal()
		tk.Update()
	}
	t.solutionMap.SetPortals(newPortals)
	t.portalList.SetPortals(newPortals)
	for guid := range newPortals {
		t.portalStateChanged(guid)
	}
}

func stringMapsAreTheSame(map1 map[string]bool, map2 map[string]bool) bool {
	for s := range map1 {
		if !map2[s] {
			return false
		}
	}
	for s := range map2 {
		if !map1[s] {
			return false
		}
	}
	return true
}

func (t *baseTab) OnSelectionChanged(selection []string) {
	selectionMap := make(map[string]bool)
	for _, guid := range selection {
		selectionMap[guid] = true
	}
	if stringMapsAreTheSame(selectionMap, t.selectedPortals) {
		return
	}
	oldSelected := t.selectedPortals
	t.selectedPortals = selectionMap
	if t.portalList != nil {
		t.portalList.SetSelectedPortals(t.selectedPortals)
	}
	if t.solutionMap != nil {
		for guid := range oldSelected {
			t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
		}
		for _, guid := range selection {
			t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
			t.solutionMap.RaisePortal(guid)
		}
	}
	if len(selection) == 1 {
		if t.portalList != nil {
			t.portalList.ScrollToPortal(selection[0])
		}
		if t.solutionMap != nil {
			t.solutionMap.ScrollToPortal(selection[0])
		}
	}
}
func (t *baseTab) OnPortalSelected(guid string) {
	t.OnSelectionChanged([]string{guid})
	if t.portalList != nil {
		t.portalList.ScrollToPortal(guid)
	}
	if t.solutionMap != nil {
		t.solutionMap.ScrollToPortal(guid)
	}
}

func (t *baseTab) portalStateChanged(guid string) {
	if t.portalList != nil {
		t.portalList.SetPortalState(guid, t.pattern.portalLabel(guid))
	}
	if t.solutionMap != nil {
		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}