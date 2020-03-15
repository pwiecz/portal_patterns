package main

import "fmt"
import "os"
import "path"
import "sort"

import "github.com/pwiecz/portal_patterns/lib"
import "github.com/pwiecz/atk/tk"
import "github.com/golang/geo/s2"

type CobwebTab struct {
	*tk.PackLayout
	add             *tk.Button
	reset           *tk.Button
	find            *tk.Button
	save            *tk.Button
	solutionLabel   *tk.Label
	progress        *tk.ProgressBar
	portalList      *PortalList
	portalScrollBar *tk.ScrollBar
	solutionMap     *SolutionMap
	portalCanvas    *tk.Canvas
	portals         map[string]lib.Portal
	solution        []lib.Portal
	length          uint16
	selectedPortals map[string]bool
	cornerPortals   map[string]bool
	disabledPortals map[string]bool
}

func NewCobwebTab(parent *Window, conf *Configuration) *CobwebTab {
	h := &CobwebTab{}
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
	h.reset.SetState(tk.StateDisable)
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
		file.WriteString(lib.CobwebDrawToolsString(h.solution))
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
	h.cornerPortals = make(map[string]bool)
	h.disabledPortals = make(map[string]bool)
	return h
}

func (h *CobwebTab) onProgress(val int, max int) {
	value := float64(val) * 1000. / float64(max)
	h.progress.SetValue(value)
	tk.Update()
}

func (h *CobwebTab) resetPortals() {
	h.portals = make(map[string]lib.Portal)
	h.selectedPortals = make(map[string]bool)
	h.cornerPortals = make(map[string]bool)
	h.disabledPortals = make(map[string]bool)
	h.reset.SetState(tk.StateDisable)
	h.find.SetState(tk.StateDisable)
	h.save.SetState(tk.StateDisable)
	if h.solutionMap != nil {
		h.solutionMap.Clear()
	}
	if h.portalList != nil {
		h.portalList.Clear()
	}
	h.solutionLabel.SetText("")
}

func (h *CobwebTab) addPortals(portals []lib.Portal) {
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
	if len(h.portals) > 0 {
		h.reset.SetState(tk.StateNormal)
	}
	if len(h.portals) >= 3 {
		h.find.SetState(tk.StateNormal)
	}
	if h.solutionMap == nil {
		h.solutionMap = NewSolutionMap(h, "Cobweb")
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

func cobwebStateToName(disabled, isCorner, selected bool) string {
	if disabled {
		return "Disabled"
	}
	if isCorner {
		return "Corner"
	}
	return "Normal"
}

func cobwebStateToColor(disabled, isCorner, selected bool) string {
	if disabled {
		if !selected {
			return "gray"
		}
		return "dark gray"
	}
	if isCorner {
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

func (h *CobwebTab) OnSelectionChanged(selection []string) {
	selectionMap := make(map[string]bool)
	for _, guid := range selection {
		selectionMap[guid] = true
	}
	if stringMapsAreTheSame(selectionMap, h.selectedPortals) {
		return
	}
	if h.solutionMap != nil {
		for portal := range h.selectedPortals {
			h.solutionMap.SetPortalColor(portal, cobwebStateToColor(h.disabledPortals[portal], h.cornerPortals[portal], selectionMap[portal]))
		}
		for _, portal := range selection {
			h.solutionMap.SetPortalColor(portal, cobwebStateToColor(h.disabledPortals[portal], h.cornerPortals[portal], true))
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
func (h *CobwebTab) OnPortalSelected(guid string) {
	h.OnSelectionChanged([]string{guid})
	if h.portalList != nil {
		h.portalList.ScrollToPortal(guid)
	}
	if h.solutionMap != nil {
		h.solutionMap.ScrollToPortal(guid)
	}
}
func (h *CobwebTab) OnPortalContextMenu(guid string, x, y int) {
	menu := NewCobwebPortalContextMenu(tk.RootWindow(), guid, h)
	tk.PopupMenu(menu.Menu, x, y)
}

func (h *CobwebTab) search() {
	if len(h.portals) < 3 {
		return
	}

	h.add.SetState(tk.StateDisable)
	h.reset.SetState(tk.StateDisable)
	h.find.SetState(tk.StateDisable)
	h.save.SetState(tk.StateDisable)
	tk.Update()
	portals := []lib.Portal{}
	corner := []int{}
	for _, portal := range h.portals {
		if !h.disabledPortals[portal.Guid] {
			portals = append(portals, portal)
			if h.cornerPortals[portal.Guid] {
				corner = append(corner, len(portals)-1)
			}
		}
	}
	h.solution = lib.LargestCobweb(portals, corner, func(val int, max int) { h.onProgress(val, max) })
	if h.solutionMap != nil {
		h.solutionMap.SetSolution([][]lib.Portal{lib.CobwebPolyline(h.solution)})
	}
	solutionText := fmt.Sprintf("Solution length: %d", len(h.solution))
	h.solutionLabel.SetText(solutionText)
	h.add.SetState(tk.StateNormal)
	h.reset.SetState(tk.StateNormal)
	h.find.SetState(tk.StateNormal)
	h.save.SetState(tk.StateNormal)
	tk.Update()
}

func (h *CobwebTab) portalStateChanged(guid string) {
	if h.portalList != nil {
		h.portalList.SetPortalState(guid, cobwebStateToName(h.disabledPortals[guid], h.cornerPortals[guid], h.selectedPortals[guid]))
	}
	if h.solutionMap != nil {
		h.solutionMap.SetPortalColor(guid, cobwebStateToColor(h.disabledPortals[guid], h.cornerPortals[guid], h.selectedPortals[guid]))
	}
}
func (h *CobwebTab) EnablePortal(guid string) {
	delete(h.disabledPortals, guid)
	h.portalStateChanged(guid)
}
func (h *CobwebTab) DisablePortal(guid string) {
	h.disabledPortals[guid] = true
	delete(h.cornerPortals, guid)
	h.portalStateChanged(guid)
}
func (h *CobwebTab) MakeCorner(guid string) {
	h.cornerPortals[guid] = true
	h.portalStateChanged(guid)
}
func (h *CobwebTab) UnmakeCorner(guid string) {
	delete(h.cornerPortals, guid)
	h.portalStateChanged(guid)
}

type CobwebPortalContextMenu struct {
	*tk.Menu
}

func NewCobwebPortalContextMenu(parent *tk.Window, guid string, h *CobwebTab) *CobwebPortalContextMenu {
	l := &CobwebPortalContextMenu{}
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
	if h.cornerPortals[guid] {
		uncornerAction := tk.NewAction("Unmake corner portal")
		uncornerAction.OnCommand(func() { h.UnmakeCorner(guid) })
		l.AddAction(uncornerAction)
	} else if !h.disabledPortals[guid] && len(h.cornerPortals) < 3 {
		cornerAction := tk.NewAction("Make corner portal")
		cornerAction.OnCommand(func() { h.MakeCorner(guid) })
		l.AddAction(cornerAction)
	}
	return l
}
