package main

import "os"
import "path"
import "sort"
import "strconv"

import "github.com/pwiecz/portal_patterns/lib"
import "github.com/pwiecz/atk/tk"
import "github.com/golang/geo/s2"

type HomogeneousTab struct {
	*tk.PackLayout
	add             *tk.Button
	reset           *tk.Button
	save            *tk.Button
	maxDepth        *tk.Entry
	pretty          *tk.CheckButton
	perfect         *tk.CheckButton
	strategy        *tk.ComboBox
	find            *tk.Button
	progress        *tk.ProgressBar
	portalList      *PortalList
	portalScrollBar *tk.ScrollBar
	solutionMap     *SolutionMap
	portalCanvas    *tk.Canvas
	portals         map[string]lib.Portal
	solution        []lib.Portal
	depth           uint16
	selectedPortals map[string]bool
	anchorPortals   map[string]bool
	disabledPortals map[string]bool
}

func NewHomogeneousTab(parent *Window, conf *Configuration) *HomogeneousTab {
	h := &HomogeneousTab{}
	h.PackLayout = tk.NewVPackLayout(parent)
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
			return
		}
		h.addPortals(portals)
	})
	h.AddWidgetEx(h.add, tk.FillNone, true, tk.AnchorWest)
	h.reset = tk.NewButton(parent, "Reset Portals")
	h.reset.OnCommand(func() {
		h.resetPortals()
	})
	h.AddWidgetEx(h.reset, tk.FillNone, true, tk.AnchorWest)
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
		file.WriteString(lib.HomogeneousDrawToolsString(h.depth, h.solution))
	})
	h.save.SetState(tk.StateDisable)
	h.AddWidgetEx(h.save, tk.FillNone, true, tk.AnchorWest)
	maxDepthBox := tk.NewHPackLayout(parent)
	maxDepthLabel := tk.NewLabel(parent, "Max depth: ")
	maxDepthBox.AddWidget(maxDepthLabel)
	h.maxDepth = tk.NewEntry(parent)
	h.maxDepth.SetText("6")
	maxDepthBox.AddWidget(h.maxDepth)
	h.AddWidget(maxDepthBox)
	h.pretty = tk.NewCheckButton(parent, "Pretty")
	h.pretty.OnCommand(func() {
		if h.pretty.IsChecked() {
			h.strategy.SetCurrentIndex(1)
		}
	})
	h.AddWidgetEx(h.pretty, tk.FillNone, true, tk.AnchorWest)
	h.perfect = tk.NewCheckButton(parent, "Perfect")
	h.AddWidgetEx(h.perfect, tk.FillNone, true, tk.AnchorWest)
	strategyBox := tk.NewHPackLayout(parent)
	strategyLabel := tk.NewLabel(parent, "Top triangle: ")
	strategyBox.AddWidget(strategyLabel)
	h.strategy = tk.NewComboBox(parent, tk.ComboBoxAttrState(tk.StateReadOnly))
	h.strategy.SetValues([]string{"Arbitrary", "Largest Area", "Smallest Area"})
	h.strategy.SetCurrentIndex(0)
	h.strategy.OnSelected(func() { h.strategy.Entry().ClearSelection() })
	strategyBox.AddWidget(h.strategy)
	h.AddWidget(strategyBox)
	h.find = tk.NewButton(parent, "Search")
	h.find.OnCommand(func() {
		h.search()
	})
	h.find.SetState(tk.StateDisable)
	h.AddWidgetEx(h.find, tk.FillNone, true, tk.AnchorWest)
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
	h.anchorPortals = make(map[string]bool)
	h.disabledPortals = make(map[string]bool)
	return h
}

func (h *HomogeneousTab) onProgress(val int, max int) {
	value := float64(val) * 1000. / float64(max)
	h.progress.SetValue(value)
	tk.Update()
}

func (h *HomogeneousTab) resetPortals() {
	h.portals = make(map[string]lib.Portal)
	h.selectedPortals = make(map[string]bool)
	h.anchorPortals = make(map[string]bool)
	h.disabledPortals = make(map[string]bool)
	h.find.SetState(tk.StateDisable)
	if h.solutionMap != nil {
		h.solutionMap.Clear()
	}
	if h.portalList != nil {
		h.portalList.Clear()
	}
}
func (h *HomogeneousTab) addPortals(portals []lib.Portal) {
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
		h.solutionMap = NewSolutionMap(h, "Homogeneous")
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

func stateToName(disabled, isAnchor, selected bool) string {
	if disabled {
		return "Disabled"
	}
	if isAnchor {
		return "Anchor"
	}
	return "Normal"
}

func stateToColor(disabled, isAnchor, selected bool) string {
	if disabled {
		if !selected {
			return "gray"
		}
		return "dark gray"
	}
	if isAnchor {
		if !selected {
			return "green"
		}
		return "dark green"
	}
	if !selected {
		return "orange"
	}
	return "red"
}

func stringMapsAreTheSame(map1 map[string]bool, map2 map[string]bool) bool {
	for s, _ := range map1 {
		if !map2[s] {
			return false
		}
	}
	for s, _ := range map2 {
		if !map1[s] {
			return false
		}
	}
	return true
}
func (h *HomogeneousTab) OnSelectionChanged(selection []string) {
	selectionMap := make(map[string]bool)
	for _, guid := range selection {
		selectionMap[guid] = true
	}
	if stringMapsAreTheSame(selectionMap, h.selectedPortals) {
		return
	}
	if h.solutionMap != nil {
		for portal, _ := range h.selectedPortals {
			h.solutionMap.SetPortalColor(portal, stateToColor(h.disabledPortals[portal], h.anchorPortals[portal], selectionMap[portal]))
		}
		for _, portal := range selection {
			h.solutionMap.SetPortalColor(portal, stateToColor(h.disabledPortals[portal], h.anchorPortals[portal], true))
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
func (h *HomogeneousTab) OnPortalSelected(guid string) {
	h.OnSelectionChanged([]string{guid})
	if h.portalList != nil {
		h.portalList.ScrollToPortal(guid)
	}
	if h.solutionMap != nil {
		h.solutionMap.ScrollToPortal(guid)
	}
}
func (h *HomogeneousTab) OnPortalContextMenu(guid string, x, y int) {
	menu := NewPortalContextMenu(tk.RootWindow(), guid, h)
	tk.PopupMenu(menu.Menu, x, y)
}

func (h *HomogeneousTab) search() {
	if len(h.portals) < 3 {
		return
	}
	options := []lib.HomogeneousOption{}
	maxDepth, err := strconv.Atoi(h.maxDepth.Text())
	if err != nil || maxDepth < 1 {
		return
	}
	options = append(options, lib.HomogeneousMaxDepth(maxDepth))
	options = append(options, lib.HomogeneousPerfect(h.perfect.IsChecked()))
	if h.strategy.CurrentIndex() == 1 {
		options = append(options, lib.HomogeneousLargestArea{})
	} else if h.strategy.CurrentIndex() == 2 {
		options = append(options, lib.HomogeneousSmallestArea{})
	}
	options = append(options, lib.HomogeneousProgressFunc(
		func(val int, max int) { h.onProgress(val, max) }))
	h.add.SetState(tk.StateDisable)
	h.reset.SetState(tk.StateDisable)
	h.maxDepth.SetState(tk.StateDisable)
	h.pretty.SetState(tk.StateDisable)
	h.perfect.SetState(tk.StateDisable)
	h.strategy.SetState(tk.StateDisable)
	h.find.SetState(tk.StateDisable)
	h.save.SetState(tk.StateDisable)
	tk.Update()
	portals := []lib.Portal{}
	anchors := []int{}
	for _, portal := range h.portals {
		if !h.disabledPortals[portal.Guid] {
			portals = append(portals, portal)
			if h.anchorPortals[portal.Guid] {
				anchors = append(anchors, len(portals)-1)
			}
		}
	}
	options = append(options, lib.HomogeneousFixedCornerIndices(anchors))
	if h.pretty.IsChecked() {
		h.solution, h.depth = lib.DeepestHomogeneous2(portals, options...)
	} else {
		h.solution, h.depth = lib.DeepestHomogeneous(portals, options...)
	}
	{
		h.solutionMap.SetSolution(lib.HomogeneousPolylines(h.depth, h.solution))
	}
	tk.MessageBox(h, "Solution found", "Found solution of depth "+strconv.Itoa(int(h.depth)), "", "", tk.MessageBoxIconInfo, tk.MessageBoxTypeOk)
	h.add.SetState(tk.StateNormal)
	h.reset.SetState(tk.StateNormal)
	h.maxDepth.SetState(tk.StateNormal)
	h.pretty.SetState(tk.StateNormal)
	h.perfect.SetState(tk.StateNormal)
	h.strategy.SetState(tk.StateReadOnly)
	h.find.SetState(tk.StateNormal)
	h.save.SetState(tk.StateNormal)
	tk.Update()
}

func (s *HomogeneousTab) portalStateChanged(guid string) {
	if s.portalList != nil {
		s.portalList.SetPortalState(guid, stateToName(s.disabledPortals[guid], s.anchorPortals[guid], s.selectedPortals[guid]))
	}
	if s.solutionMap != nil {
		s.solutionMap.SetPortalColor(guid, stateToColor(s.disabledPortals[guid], s.anchorPortals[guid], s.selectedPortals[guid]))
	}
}
func (s *HomogeneousTab) EnablePortal(guid string) {
	delete(s.disabledPortals, guid)
	s.portalStateChanged(guid)
}
func (s *HomogeneousTab) DisablePortal(guid string) {
	s.disabledPortals[guid] = true
	delete(s.anchorPortals, guid)
	s.portalStateChanged(guid)
}
func (s *HomogeneousTab) MakeAnchor(guid string) {
	s.anchorPortals[guid] = true
	s.portalStateChanged(guid)
}
func (s *HomogeneousTab) UnmakeAnchor(guid string) {
	delete(s.anchorPortals, guid)
	s.portalStateChanged(guid)
}

type PortalContextMenu struct {
	*tk.Menu
}

func NewPortalContextMenu(parent *tk.Window, guid string, h *HomogeneousTab) *PortalContextMenu {
	l := &PortalContextMenu{}
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
	if h.anchorPortals[guid] {
		unanchorAction := tk.NewAction("Unmake anchor")
		unanchorAction.OnCommand(func() { h.UnmakeAnchor(guid) })
		l.AddAction(unanchorAction)
	} else if !h.disabledPortals[guid] && len(h.anchorPortals) < 3 {
		anchorAction := tk.NewAction("Make anchor")
		anchorAction.OnCommand(func() { h.MakeAnchor(guid) })
		l.AddAction(anchorAction)
	}
	return l
}
