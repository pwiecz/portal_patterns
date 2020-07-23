package main

import "fmt"

import "github.com/pwiecz/portal_patterns/gui/osm"
import "github.com/pwiecz/portal_patterns/lib"
import "github.com/pwiecz/atk/tk"

type cobwebTab struct {
	*baseTab
	solution      []lib.Portal
	cornerPortals map[string]bool
}

func NewCobwebTab(parent *Window, conf *Configuration, tileFetcher *osm.MapTiles) *cobwebTab {
	t := &cobwebTab{}
	t.baseTab = NewBaseTab(parent, "Cobweb", conf, tileFetcher)
	t.pattern = t
	addResetBox := tk.NewHPackLayout(parent)
	addResetBox.AddWidget(t.add)
	addResetBox.AddWidget(t.reset)
	t.AddWidget(addResetBox)
	solutionBox := tk.NewHPackLayout(parent)
	solutionBox.AddWidget(t.find)
	solutionBox.AddWidget(t.save)
	solutionBox.AddWidget(t.copy)
	solutionBox.AddWidget(t.solutionLabel)
	t.AddWidget(solutionBox)
	t.AddWidgetEx(t.progress, tk.FillBoth, true, tk.AnchorWest)
	t.AddWidgetEx(t.portalList, tk.FillBoth, true, tk.AnchorWest)

	t.cornerPortals = make(map[string]bool)
	return t
}

func (t *cobwebTab) onReset() {
	t.cornerPortals = make(map[string]bool)
}

func (t *cobwebTab) portalLabel(guid string) string {
	if t.disabledPortals[guid] {
		return "Disabled"
	}
	if t.cornerPortals[guid] {
		return "Corner"
	}
	return "Normal"
}

func (t *cobwebTab) portalColor(guid string) string {
	if t.disabledPortals[guid] {
		if !t.selectedPortals[guid] {
			return "gray"
		}
		return "dark gray"
	}
	if t.cornerPortals[guid] {
		if !t.selectedPortals[guid] {
			return "green"
		}
		return "dark green"
	}
	if !t.selectedPortals[guid] {
		return "orange"
	}
	return "red"
}

func (t *cobwebTab) onPortalContextMenu(guid string, x, y int) {
	menu := NewCobwebPortalContextMenu(tk.RootWindow(), guid, t)
	tk.PopupMenu(menu.Menu, x, y)
}

func (t *cobwebTab) search() {
	if len(t.portals) < 3 {
		return
	}

	t.add.SetState(tk.StateDisable)
	t.reset.SetState(tk.StateDisable)
	t.find.SetState(tk.StateDisable)
	t.save.SetState(tk.StateDisable)
	t.copy.SetState(tk.StateDisable)
	tk.Update()
	portals := []lib.Portal{}
	corner := []int{}
	for _, portal := range t.portals {
		if !t.disabledPortals[portal.Guid] {
			portals = append(portals, portal)
			if t.cornerPortals[portal.Guid] {
				corner = append(corner, len(portals)-1)
			}
		}
	}
	t.solution = lib.LargestCobweb(portals, corner, func(val int, max int) { t.onProgress(val, max) })
	if t.solutionMap != nil {
		t.solutionMap.SetSolution([][]lib.Portal{lib.CobwebPolyline(t.solution)})
	}
	solutionText := fmt.Sprintf("Solution length: %d", len(t.solution))
	t.solutionLabel.SetText(solutionText)
	t.add.SetState(tk.StateNormal)
	t.reset.SetState(tk.StateNormal)
	t.find.SetState(tk.StateNormal)
	t.save.SetState(tk.StateNormal)
	t.copy.SetState(tk.StateNormal)
	tk.Update()
}

func (t *cobwebTab) solutionString() string {
	return lib.CobwebDrawToolsString(t.solution)
}
func (t *cobwebTab) EnablePortal(guid string) {
	delete(t.disabledPortals, guid)
	t.portalStateChanged(guid)
}
func (t *cobwebTab) DisablePortal(guid string) {
	t.disabledPortals[guid] = true
	delete(t.cornerPortals, guid)
	t.portalStateChanged(guid)
}
func (t *cobwebTab) MakeCorner(guid string) {
	t.cornerPortals[guid] = true
	t.portalStateChanged(guid)
}
func (t *cobwebTab) UnmakeCorner(guid string) {
	delete(t.cornerPortals, guid)
	t.portalStateChanged(guid)
}

type cobwebPortalContextMenu struct {
	*tk.Menu
}

func NewCobwebPortalContextMenu(parent *tk.Window, guid string, t *cobwebTab) *cobwebPortalContextMenu {
	l := &cobwebPortalContextMenu{}
	l.Menu = tk.NewMenu(parent)
	if t.disabledPortals[guid] {
		enableAction := tk.NewAction("Enable")
		enableAction.OnCommand(func() { t.EnablePortal(guid) })
		l.AddAction(enableAction)
	} else {
		disableAction := tk.NewAction("Disable")
		disableAction.OnCommand(func() { t.DisablePortal(guid) })
		l.AddAction(disableAction)
	}
	if t.cornerPortals[guid] {
		uncornerAction := tk.NewAction("Unmake corner portal")
		uncornerAction.OnCommand(func() { t.UnmakeCorner(guid) })
		l.AddAction(uncornerAction)
	} else if !t.disabledPortals[guid] && len(t.cornerPortals) < 3 {
		cornerAction := tk.NewAction("Make corner portal")
		cornerAction.OnCommand(func() { t.MakeCorner(guid) })
		l.AddAction(cornerAction)
	}
	return l
}
