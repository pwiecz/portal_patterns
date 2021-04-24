package main

// import (
// 	"fmt"
// 	"runtime"

// 	"github.com/pwiecz/atk/tk"
// 	"github.com/pwiecz/portal_patterns/configuration"
// 	"github.com/pwiecz/portal_patterns/gui/osm"
// 	"github.com/pwiecz/portal_patterns/lib"
// )

// type droneFlightTab struct {
// 	*baseTab
// 	useLongJumps   *tk.CheckButton
// 	optimizeFor    *tk.ComboBox
// 	solution, keys []lib.Portal
// 	startPortal    string
// 	endPortal      string
// }

// func NewDroneFlightTab(parent *Window, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *droneFlightTab {
// 	t := &droneFlightTab{}
// 	t.baseTab = NewBaseTab(parent, "Drone Flight", conf, tileFetcher)
// 	t.pattern = t
// 	addResetBox := tk.NewHPackLayout(parent)
// 	addResetBox.AddWidget(t.add)
// 	addResetBox.AddWidget(t.reset)
// 	t.AddWidget(addResetBox)
// 	t.useLongJumps = tk.NewCheckButton(parent, "Use long jumps (key needed)")
// 	t.useLongJumps.SetChecked(true)
// 	t.AddWidgetEx(t.useLongJumps, tk.FillNone, true, tk.AnchorWest)
// 	optimizeForBox := tk.NewHPackLayout(parent)
// 	optimizeForLabel := tk.NewLabel(parent, "Optimize for: ")
// 	optimizeForBox.AddWidget(optimizeForLabel)
// 	t.optimizeFor = tk.NewComboBox(parent, tk.ComboBoxAttrState(tk.StateReadOnly))
// 	t.optimizeFor.SetValues([]string{"Least keys needed", "Least jumps"})
// 	t.optimizeFor.SetCurrentIndex(0)
// 	t.optimizeFor.OnSelected(func() { t.optimizeFor.Entry().ClearSelection() })
// 	optimizeForBox.AddWidget(t.optimizeFor)
// 	t.AddWidget(optimizeForBox)
// 	solutionBox := tk.NewHPackLayout(parent)
// 	solutionBox.AddWidget(t.find)
// 	solutionBox.AddWidget(t.save)
// 	solutionBox.AddWidget(t.copy)
// 	solutionBox.AddWidget(t.solutionLabel)
// 	t.AddWidget(solutionBox)
// 	t.AddWidgetEx(t.progress, tk.FillBoth, true, tk.AnchorWest)
// 	t.AddWidgetEx(t.portalList, tk.FillBoth, true, tk.AnchorWest)

// 	return t
// }

// func (t *droneFlightTab) onReset() {
// 	t.startPortal = ""
// 	t.endPortal = ""
// }

// func (t *droneFlightTab) portalLabel(guid string) string {
// 	if t.disabledPortals[guid] {
// 		return "Disabled"
// 	}
// 	if t.startPortal == guid {
// 		return "Start"
// 	}
// 	if t.endPortal == guid {
// 		return "End"
// 	}
// 	return "Normal"
// }

// func (t *droneFlightTab) portalColor(guid string) string {
// 	if t.disabledPortals[guid] {
// 		if !t.selectedPortals[guid] {
// 			return "gray"
// 		}
// 		return "dark gray"
// 	}
// 	if t.startPortal == guid {
// 		if !t.selectedPortals[guid] {
// 			return "green"
// 		}
// 		return "dark green"
// 	}
// 	if t.endPortal == guid {
// 		if !t.selectedPortals[guid] {
// 			return "yellow"
// 		}
// 		return "khaki"
// 	}
// 	if !t.selectedPortals[guid] {
// 		return "orange"
// 	}
// 	return "red"
// }

// func (t *droneFlightTab) onPortalContextMenu(guid string, x, y int) {
// 	menu := NewDroneFlightPortalContextMenu(tk.RootWindow(), guid, t)
// 	tk.PopupMenu(menu.Menu, x, y)
// }

// func (t *droneFlightTab) search() {
// 	if len(t.portals) < 3 {
// 		return
// 	}

// 	t.add.SetState(tk.StateDisable)
// 	t.reset.SetState(tk.StateDisable)
// 	t.find.SetState(tk.StateDisable)
// 	t.save.SetState(tk.StateDisable)
// 	t.copy.SetState(tk.StateDisable)
// 	tk.Update()
// 	portals := []lib.Portal{}
// 	options := []lib.DroneFlightOption{lib.DroneFlightNumWorkers(runtime.GOMAXPROCS(0))}
// 	for _, portal := range t.portals {
// 		if !t.disabledPortals[portal.Guid] {
// 			portals = append(portals, portal)
// 			if t.startPortal == portal.Guid {
// 				options = append(options, lib.DroneFlightStartPortalIndex(len(portals)-1))
// 			}
// 			if t.endPortal == portal.Guid {
// 				options = append(options, lib.DroneFlightEndPortalIndex(len(portals)-1))
// 			}
// 		}
// 	}
// 	options = append(options, lib.DroneFlightUseLongJumps(t.useLongJumps.IsChecked()))
// 	options = append(options, lib.DroneFlightProgressFunc(
// 		func(val int, max int) { t.onProgress(val, max) }))
// 	if t.optimizeFor.CurrentIndex() == 1 {
// 		options = append(options, lib.DroneFlightLeastJumps{})
// 	}
// 	t.solution, t.keys = lib.LongestDroneFlight(portals, options...)
// 	if t.solutionMap != nil {
// 		t.solutionMap.SetSolution([][]lib.Portal{t.solution})
// 	}
// 	distance := t.solution[0].LatLng.Distance(t.solution[len(t.solution)-1].LatLng) * lib.RadiansToMeters
// 	solutionText := fmt.Sprintf("Flight distance: %.1fm, keys needed: %d", distance, len(t.keys))
// 	t.solutionLabel.SetText(solutionText)
// 	t.add.SetState(tk.StateNormal)
// 	t.reset.SetState(tk.StateNormal)
// 	t.find.SetState(tk.StateNormal)
// 	t.save.SetState(tk.StateNormal)
// 	t.copy.SetState(tk.StateNormal)
// 	tk.Update()
// }

// func (t *droneFlightTab) solutionString() string {
// 	s := fmt.Sprintf("[%s", lib.PolylineFromPortalList(t.solution))
// 	if len(t.keys) > 0 {
// 		s += fmt.Sprintf(",%s", lib.MarkersFromPortalList(t.keys))
// 	}
// 	return s + "]"
// }
// func (t *droneFlightTab) EnablePortal(guid string) {
// 	delete(t.disabledPortals, guid)
// 	t.portalStateChanged(guid)
// }
// func (t *droneFlightTab) DisablePortal(guid string) {
// 	t.disabledPortals[guid] = true
// 	if t.startPortal == guid {
// 		t.startPortal = ""
// 	}
// 	if t.endPortal == guid {
// 		t.endPortal = ""
// 	}
// 	t.portalStateChanged(guid)
// }

// func (t *droneFlightTab) MakeStart(guid string) {
// 	if t.startPortal != "" {
// 		oldStartGuid := t.startPortal
// 		t.startPortal = ""
// 		t.portalStateChanged(oldStartGuid)
// 	}
// 	t.startPortal = guid
// 	if t.endPortal == guid {
// 		t.endPortal = ""
// 	}
// 	t.portalStateChanged(guid)
// }
// func (t *droneFlightTab) UnmakeStart(guid string) {
// 	t.startPortal = ""
// 	t.portalStateChanged(guid)
// }
// func (t *droneFlightTab) MakeEnd(guid string) {
// 	if t.endPortal != "" {
// 		oldEndGuid := t.endPortal
// 		t.endPortal = ""
// 		t.portalStateChanged(oldEndGuid)
// 	}
// 	t.endPortal = guid
// 	if t.startPortal == guid {
// 		t.startPortal = ""
// 	}
// 	t.portalStateChanged(guid)
// }
// func (t *droneFlightTab) UnmakeEnd(guid string) {
// 	t.endPortal = ""
// 	t.portalStateChanged(guid)
// }

// type droneFlightPortalContextMenu struct {
// 	*tk.Menu
// }

// func NewDroneFlightPortalContextMenu(parent *tk.Window, guid string, t *droneFlightTab) *droneFlightPortalContextMenu {
// 	l := &droneFlightPortalContextMenu{}
// 	l.Menu = tk.NewMenu(parent)
// 	if t.disabledPortals[guid] {
// 		enableAction := tk.NewAction("Enable")
// 		enableAction.OnCommand(func() { t.EnablePortal(guid) })
// 		l.AddAction(enableAction)
// 	} else {
// 		disableAction := tk.NewAction("Disable")
// 		disableAction.OnCommand(func() { t.DisablePortal(guid) })
// 		l.AddAction(disableAction)
// 	}
// 	if t.startPortal == guid {
// 		unstartAction := tk.NewAction("Unmake start portal")
// 		unstartAction.OnCommand(func() { t.UnmakeStart(guid) })
// 		l.AddAction(unstartAction)
// 	} else if !t.disabledPortals[guid] {
// 		startAction := tk.NewAction("Make start portal")
// 		startAction.OnCommand(func() { t.MakeStart(guid) })
// 		l.AddAction(startAction)
// 	}
// 	if t.endPortal == guid {
// 		unendAction := tk.NewAction("Unmake end portal")
// 		unendAction.OnCommand(func() { t.UnmakeEnd(guid) })
// 		l.AddAction(unendAction)
// 	} else if !t.disabledPortals[guid] {
// 		endAction := tk.NewAction("Make end portal")
// 		endAction.OnCommand(func() { t.MakeEnd(guid) })
// 		l.AddAction(endAction)
// 	}

// 	return l
// }
