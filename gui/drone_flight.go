package main

import "fmt"

import "github.com/pwiecz/portal_patterns/lib"
import "github.com/pwiecz/atk/tk"

type droneFlightTab struct {
	*baseTab
	solution    []lib.Portal
	startPortal string
	endPortal   string
}

func NewDroneFlightTab(parent *Window, conf *Configuration) *droneFlightTab {
	t := &droneFlightTab{}
	t.baseTab = NewBaseTab(parent, "Drone Flight", conf)
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

	return t
}

func (t *droneFlightTab) onReset() {
	t.startPortal = ""
	t.endPortal = ""
}

func (t *droneFlightTab) portalLabel(guid string) string {
	if t.disabledPortals[guid] {
		return "Disabled"
	}
	if t.startPortal == guid {
		return "Start"
	}
	if t.endPortal == guid {
		return "End"
	}
	return "Normal"
}

func (t *droneFlightTab) portalColor(guid string) string {
	if t.disabledPortals[guid] {
		if !t.selectedPortals[guid] {
			return "gray"
		}
		return "dark gray"
	}
	if t.startPortal == guid {
		if !t.selectedPortals[guid] {
			return "green"
		}
		return "dark green"
	}
	if t.endPortal == guid {
		if !t.selectedPortals[guid] {
			return "yellow"
		}
		return "khaki"
	}
	if !t.selectedPortals[guid] {
		return "orange"
	}
	return "red"
}

func (t *droneFlightTab) onPortalContextMenu(guid string, x, y int) {
	menu := NewDroneFlightPortalContextMenu(tk.RootWindow(), guid, t)
	tk.PopupMenu(menu.Menu, x, y)
}

func (t *droneFlightTab) search() {
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
	startPortal, endPortal := -1, -1
	for _, portal := range t.portals {
		if !t.disabledPortals[portal.Guid] {
			portals = append(portals, portal)
			if t.startPortal == portal.Guid {
				startPortal = len(portals) - 1
			}
			if t.endPortal == portal.Guid {
				endPortal = len(portals) - 1
			}
		}
	}
	var distance float64
	t.solution, distance = lib.LongestDroneFlight(portals, startPortal, endPortal, func(val int, max int) { t.onProgress(val, max) })
	if t.solutionMap != nil {
		t.solutionMap.SetSolution([][]lib.Portal{t.solution})
	}
	solutionText := fmt.Sprintf("Flight distance: %f", distance)
	t.solutionLabel.SetText(solutionText)
	t.add.SetState(tk.StateNormal)
	t.reset.SetState(tk.StateNormal)
	t.find.SetState(tk.StateNormal)
	t.save.SetState(tk.StateNormal)
	t.copy.SetState(tk.StateNormal)
	tk.Update()
}

func (t *droneFlightTab) solutionString() string {
	return fmt.Sprintf("[%s]", lib.PolylineFromPortalList(t.solution))
}
func (t *droneFlightTab) EnablePortal(guid string) {
	delete(t.disabledPortals, guid)
	t.portalStateChanged(guid)
}
func (t *droneFlightTab) DisablePortal(guid string) {
	t.disabledPortals[guid] = true
	if t.startPortal == guid {
		t.startPortal = ""
	}
	if t.endPortal == guid {
		t.endPortal = ""
	}
	t.portalStateChanged(guid)
}

func (t *droneFlightTab) MakeStart(guid string) {
	if t.startPortal != "" {
		oldStartGuid := t.startPortal
		t.startPortal = ""
		t.portalStateChanged(oldStartGuid)
	}
	t.startPortal = guid
	if t.endPortal == guid {
		t.endPortal = ""
	}
	t.portalStateChanged(guid)
}
func (t *droneFlightTab) UnmakeStart(guid string) {
	t.startPortal = ""
	t.portalStateChanged(guid)
}
func (t *droneFlightTab) MakeEnd(guid string) {
	if t.endPortal != "" {
		oldEndGuid := t.endPortal
		t.endPortal = ""
		t.portalStateChanged(oldEndGuid)
	}
	t.endPortal = guid
	if t.startPortal == guid {
		t.startPortal = ""
	}
	t.portalStateChanged(guid)
}
func (t *droneFlightTab) UnmakeEnd(guid string) {
	t.endPortal = ""
	t.portalStateChanged(guid)
}

type droneFlightPortalContextMenu struct {
	*tk.Menu
}

func NewDroneFlightPortalContextMenu(parent *tk.Window, guid string, t *droneFlightTab) *droneFlightPortalContextMenu {
	l := &droneFlightPortalContextMenu{}
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
	if t.startPortal == guid {
		unstartAction := tk.NewAction("Unmake start portal")
		unstartAction.OnCommand(func() { t.UnmakeStart(guid) })
		l.AddAction(unstartAction)
	} else if !t.disabledPortals[guid] {
		startAction := tk.NewAction("Make start portal")
		startAction.OnCommand(func() { t.MakeStart(guid) })
		l.AddAction(startAction)
	}
	if t.endPortal == guid {
		unendAction := tk.NewAction("Unmake end portal")
		unendAction.OnCommand(func() { t.UnmakeEnd(guid) })
		l.AddAction(unendAction)
	} else if !t.disabledPortals[guid] {
		endAction := tk.NewAction("Make end portal")
		endAction.OnCommand(func() { t.MakeEnd(guid) })
		l.AddAction(endAction)
	}

	return l
}
