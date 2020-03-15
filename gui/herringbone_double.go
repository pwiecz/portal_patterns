package main

import "fmt"
import "os"
import "path"
import "runtime"
import "sort"

import "github.com/pwiecz/portal_patterns/lib"
import "github.com/pwiecz/atk/tk"
import "github.com/golang/geo/s2"

type DoubleHerringboneTab struct {
	*tk.PackLayout
	add                  *tk.Button
	reset                *tk.Button
	find                 *tk.Button
	save                 *tk.Button
	solutionLabel        *tk.Label
	progress             *tk.ProgressBar
	portalList           *PortalList
	portalScrollBar      *tk.ScrollBar
	solutionMap          *SolutionMap
	portalCanvas         *tk.Canvas
	portals              map[string]lib.Portal
	b0, b1               lib.Portal
	solution0, solution1 []lib.Portal
	length               uint16
	selectedPortals      map[string]bool
	basePortals          map[string]bool
	disabledPortals      map[string]bool
}

func NewDoubleHerringboneTab(parent *Window, conf *Configuration) *DoubleHerringboneTab {
	h := &DoubleHerringboneTab{}
	h.PackLayout = tk.NewVPackLayout(parent)
	addResetBox := tk.NewHPackLayout(parent)
	h.add = tk.NewButton(parent, "Add Portals")
	h.add.OnCommand(func() {
		filename, err := tk.GetOpenFile(parent, "Choose portals file",
			[]tk.FileType{
				tk.FileType{Info: "JSON file", Ext: ".json"},
				tk.FileType{Info: "CSV file", Ext: ".csv"},
			}, conf.PortalsDirectory, "")
		if err != nil || filename == "" {
			return
		}
		portalsDir, _ := path.Split(filename)
		conf.PortalsDirectory = portalsDir
		SaveConfiguration(conf)
		portals, err := lib.ParseFile(filename)
		if err != nil {
			tk.MessageBox(h, "Could not read file", fmt.Sprintf("Error reading file:\n%v", err),
				"", "", tk.MessageBoxIconError, tk.MessageBoxTypeOk)
			return
		}
		h.addPortals(portals)
	})
	addResetBox.AddWidget(h.add)
	h.reset = tk.NewButton(parent, "Reset Portals")
	h.reset.OnCommand(func() {
		h.resetPortals()
	})
	addResetBox.AddWidget(h.reset)
	h.AddWidget(addResetBox)
	solutionBox := tk.NewHPackLayout(parent)
	h.find = tk.NewButton(parent, "Search")
	h.find.OnCommand(func() {
		h.search()
	})
	h.find.SetState(tk.StateDisable)
	solutionBox.AddWidget(h.find)
	h.save = tk.NewButton(parent, "Save Solution")
	h.save.OnCommand(func() {
		filename, err := tk.GetSaveFile(parent, "Select file for solution", true, ".json", []tk.FileType{}, conf.PortalsDirectory, "")
		if err != nil || filename == "" {
			return
		}
		file, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		file.WriteString(lib.DoubleHerringboneDrawToolsString(h.b0, h.b1, h.solution0, h.solution1))
	})
	h.save.SetState(tk.StateDisable)
	solutionBox.AddWidget(h.save)
	h.solutionLabel = tk.NewLabel(parent, "")
	solutionBox.AddWidget(h.solutionLabel)
	h.AddWidget(solutionBox)
	h.progress = tk.NewProgressBar(parent, tk.Horizontal, tk.ProgressBarAttrMaximum(1000))
	h.progress.SetDeterminateMode(true)
	h.AddWidgetEx(h.progress, tk.FillBoth, true, tk.AnchorWest)
	h.portalList = NewPortalList(parent)
	h.AddWidgetEx(h.portalList, tk.FillBoth, true, tk.AnchorWest)
	h.portalList.OnPortalRightClick(func(guid string, x, y int) {
		h.OnPortalContextMenu(guid, x, y)
	})
	h.portalList.OnSelectionChanged(func() {
		selectedPortals := h.portalList.SelectedPortals()
		h.OnSelectionChanged(selectedPortals)
	})

	h.portals = make(map[string]lib.Portal)
	h.selectedPortals = make(map[string]bool)
	h.basePortals = make(map[string]bool)
	h.disabledPortals = make(map[string]bool)
	return h
}

func (h *DoubleHerringboneTab) onProgress(val int, max int) {
	value := float64(val) * 1000. / float64(max)
	h.progress.SetValue(value)
	tk.Update()
}

func (h *DoubleHerringboneTab) resetPortals() {
	h.portals = make(map[string]lib.Portal)
	h.selectedPortals = make(map[string]bool)
	h.basePortals = make(map[string]bool)
	h.disabledPortals = make(map[string]bool)
	h.find.SetState(tk.StateDisable)
	if h.solutionMap != nil {
		h.solutionMap.Clear()
	}
	if h.portalList != nil {
		h.portalList.Clear()
	}
	h.solutionLabel.SetText("")
}

func (h *DoubleHerringboneTab) addPortals(portals []lib.Portal) {
	newPortals := make(map[string]lib.Portal)
	for guid, portal := range h.portals {
		newPortals[guid] = portal
	}
	for _, portal := range portals {
		if existing, ok := h.portals[portal.Guid]; ok {
			if existing.LatLng.Lat != portal.LatLng.Lat ||
				existing.LatLng.Lng != portal.LatLng.Lng {
				if existing.Name == portal.Name {
					tk.MessageBox(h, "Conflicting portals", "Portal with guid \""+portal.Guid+"\" already loaded with different location",
						portal.Name+"\n"+portal.LatLng.String()+" vs "+existing.LatLng.String(), "", tk.MessageBoxIconWarning, tk.MessageBoxTypeOk)
					return
				}
				tk.MessageBox(h, "Conflicting portals", "Portal with guid \""+portal.Guid+"\" already loaded with different name and location",
					portal.Name+" vs "+existing.Name+"\n"+portal.LatLng.String()+" vs "+existing.LatLng.String(), "", tk.MessageBoxIconWarning, tk.MessageBoxTypeOk)
				return
			}
		}
		newPortals[portal.Guid] = portal
	}

	newPortalList := []lib.Portal{}
	for _, portal := range newPortals {
		newPortalList = append(newPortalList, portal)
	}
	for i, p0 := range newPortalList {
		for _, p1 := range newPortalList[i+1:] {
			if s2.PointFromLatLng(p0.LatLng).Distance(s2.PointFromLatLng(p1.LatLng)) >= 1. {
				tk.MessageBox(h, "Too distant portals", "Distances between portals are too large", "E.g. "+p0.Name+" and "+p1.Name, "", tk.MessageBoxIconWarning, tk.MessageBoxTypeOk)

				return
			}
		}

	}
	sort.Slice(newPortalList, func(i, j int) bool {
		return newPortalList[i].Name < newPortalList[j].Name
	})
	h.portals = newPortals
	if len(h.portals) >= 3 {
		h.find.SetState(tk.StateNormal)
	}
	if h.solutionMap == nil {
		h.solutionMap = NewSolutionMap(h, "Double herringbone")
		h.solutionMap.OnClose(func() bool {
			h.solutionMap = nil
			return true
		})
		h.solutionMap.OnPortalLeftClick(func(guid string) {
			h.OnPortalSelected(guid)
		})
		h.solutionMap.OnPortalRightClick(func(guid string, x, y int) {
			h.OnPortalContextMenu(guid, x, y)
		})
		h.solutionMap.ShowNormal()
		tk.Update()
	}
	h.solutionMap.SetPortals(newPortalList)
	h.portalList.SetPortals(newPortalList)
	for _, portal := range newPortalList {
		h.portalStateChanged(portal.Guid)
	}
}

func (h *DoubleHerringboneTab) OnSelectionChanged(selection []string) {
	selectionMap := make(map[string]bool)
	for _, guid := range selection {
		selectionMap[guid] = true
	}
	if stringMapsAreTheSame(selectionMap, h.selectedPortals) {
		return
	}
	if h.solutionMap != nil {
		for portal, _ := range h.selectedPortals {
			h.solutionMap.SetPortalColor(portal, herringboneStateToColor(h.disabledPortals[portal], h.basePortals[portal], selectionMap[portal]))
		}
		for _, portal := range selection {
			h.solutionMap.SetPortalColor(portal, herringboneStateToColor(h.disabledPortals[portal], h.basePortals[portal], true))
			h.solutionMap.RaisePortal(portal)
		}
	}
	h.selectedPortals = selectionMap
	if h.portalList != nil {
		h.portalList.SetSelectedPortals(h.selectedPortals)
	}
	if len(selection) == 1 {
		if h.portalList != nil {
			h.portalList.ScrollToPortal(selection[0])
		}
		if h.solutionMap != nil {
			h.solutionMap.ScrollToPortal(selection[0])
		}
	}
}
func (h *DoubleHerringboneTab) OnPortalSelected(guid string) {
	h.OnSelectionChanged([]string{guid})
	if h.portalList != nil {
		h.portalList.ScrollToPortal(guid)
	}
	if h.solutionMap != nil {
		h.solutionMap.ScrollToPortal(guid)
	}
}
func (h *DoubleHerringboneTab) OnPortalContextMenu(guid string, x, y int) {
	menu := NewDoubleHerringbonePortalContextMenu(tk.RootWindow(), guid, h)
	tk.PopupMenu(menu.Menu, x, y)
}

func (h *DoubleHerringboneTab) search() {
	if len(h.portals) < 3 {
		return
	}

	h.add.SetState(tk.StateDisable)
	h.reset.SetState(tk.StateDisable)
	h.find.SetState(tk.StateDisable)
	h.save.SetState(tk.StateDisable)
	tk.Update()
	portals := []lib.Portal{}
	base := []int{}
	for _, portal := range h.portals {
		if !h.disabledPortals[portal.Guid] {
			portals = append(portals, portal)
			if h.basePortals[portal.Guid] {
				base = append(base, len(portals)-1)
			}
		}
	}
	h.b0, h.b1, h.solution0, h.solution1 = lib.LargestDoubleHerringbone(portals, base, runtime.GOMAXPROCS(0), func(val int, max int) { h.onProgress(val, max) })
	if h.solutionMap != nil {
		h.solutionMap.SetSolution([][]lib.Portal{lib.DoubleHerringbonePolyline(h.b0, h.b1, h.solution0, h.solution1)})
	}
	solutionText := fmt.Sprintf("Solution length: %d + %d", len(h.solution0), len(h.solution1))
	h.solutionLabel.SetText(solutionText)
	h.add.SetState(tk.StateNormal)
	h.reset.SetState(tk.StateNormal)
	h.find.SetState(tk.StateNormal)
	h.save.SetState(tk.StateNormal)
	tk.Update()
}

func (s *DoubleHerringboneTab) portalStateChanged(guid string) {
	if s.portalList != nil {
		s.portalList.SetPortalState(guid, herringboneStateToName(s.disabledPortals[guid], s.basePortals[guid], s.selectedPortals[guid]))
	}
	if s.solutionMap != nil {
		s.solutionMap.SetPortalColor(guid, herringboneStateToColor(s.disabledPortals[guid], s.basePortals[guid], s.selectedPortals[guid]))
	}
}
func (s *DoubleHerringboneTab) EnablePortal(guid string) {
	delete(s.disabledPortals, guid)
	s.portalStateChanged(guid)
}
func (s *DoubleHerringboneTab) DisablePortal(guid string) {
	s.disabledPortals[guid] = true
	delete(s.basePortals, guid)
	s.portalStateChanged(guid)
}
func (s *DoubleHerringboneTab) MakeBase(guid string) {
	s.basePortals[guid] = true
	s.portalStateChanged(guid)
}
func (s *DoubleHerringboneTab) UnmakeBase(guid string) {
	delete(s.basePortals, guid)
	s.portalStateChanged(guid)
}

type DoubleHerringbonePortalContextMenu struct {
	*tk.Menu
}

func NewDoubleHerringbonePortalContextMenu(parent *tk.Window, guid string, h *DoubleHerringboneTab) *DoubleHerringbonePortalContextMenu {
	l := &DoubleHerringbonePortalContextMenu{}
	l.Menu = tk.NewMenu(parent)
	if h.disabledPortals[guid] {
		enableAction := tk.NewAction("Enable")
		enableAction.OnCommand(func() { h.EnablePortal(guid) })
		l.AddAction(enableAction)
	} else {
		disableAction := tk.NewAction("Disable")
		disableAction.OnCommand(func() { h.DisablePortal(guid) })
		l.AddAction(disableAction)
	}
	if h.basePortals[guid] {
		unbaseAction := tk.NewAction("Unmake base portal")
		unbaseAction.OnCommand(func() { h.UnmakeBase(guid) })
		l.AddAction(unbaseAction)
	} else if !h.disabledPortals[guid] && len(h.basePortals) < 2 {
		baseAction := tk.NewAction("Make base portal")
		baseAction.OnCommand(func() { h.MakeBase(guid) })
		l.AddAction(baseAction)
	}
	return l
}
