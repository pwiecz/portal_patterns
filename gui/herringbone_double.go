package main

// import "fmt"
// import "runtime"

// import "github.com/pwiecz/portal_patterns/gui/osm"
// import "github.com/pwiecz/portal_patterns/configuration"
// import "github.com/pwiecz/portal_patterns/lib"
// import "github.com/pwiecz/atk/tk"

// type doubleHerringboneTab struct {
// 	*baseTab
// 	b0, b1               lib.Portal
// 	solution0, solution1 []lib.Portal
// 	basePortals          map[string]bool
// }

// func NewDoubleHerringboneTab(parent tk.Widget, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *doubleHerringboneTab {
// 	t := &doubleHerringboneTab{}
// 	t.baseTab = NewBaseTab(parent, "Double Herringbone", conf, tileFetcher)
// 	t.pattern = t
// 	addResetBox := tk.NewHPackLayout(parent)
// 	addResetBox.AddWidget(t.add)
// 	addResetBox.AddWidget(t.reset)
// 	t.AddWidget(addResetBox)
// 	solutionBox := tk.NewHPackLayout(parent)
// 	solutionBox.AddWidget(t.find)
// 	solutionBox.AddWidget(t.save)
// 	solutionBox.AddWidget(t.copy)
// 	solutionBox.AddWidget(t.solutionLabel)
// 	t.AddWidget(solutionBox)
// 	t.AddWidgetEx(t.progress, tk.FillBoth, true, tk.AnchorWest)
// 	t.AddWidgetEx(t.portalList, tk.FillBoth, true, tk.AnchorWest)

// 	t.basePortals = make(map[string]bool)
// 	return t
// }

// func (t *doubleHerringboneTab) onReset() {
// 	t.basePortals = make(map[string]bool)
// }

// func (t *doubleHerringboneTab) portalLabel(guid string) string {
// 	if t.disabledPortals[guid] {
// 		return "Disabled"
// 	}
// 	if t.basePortals[guid] {
// 		return "Base"
// 	}
// 	return "Normal"
// }

// func (t *doubleHerringboneTab) portalColor(guid string) string {
// 	if t.disabledPortals[guid] {
// 		if !t.selectedPortals[guid] {
// 			return "gray"
// 		}
// 		return "dark gray"
// 	}
// 	if t.basePortals[guid] {
// 		if !t.selectedPortals[guid] {
// 			return "green"
// 		}
// 		return "dark green"
// 	}
// 	if !t.selectedPortals[guid] {
// 		return "orange"
// 	}
// 	return "red"
// }

// func (t *doubleHerringboneTab) onPortalContextMenu(guid string, x, y int) {
// 	menu := NewDoubleHerringbonePortalContextMenu(tk.RootWindow(), guid, t)
// 	tk.PopupMenu(menu.Menu, x, y)
// }

// func (t *doubleHerringboneTab) search() {
// 	if len(t.portals) < 3 {
// 		return
// 	}

// 	t.add.SetState(tk.StateDisable)
// 	t.reset.SetState(tk.StateDisable)
// 	t.find.SetState(tk.StateDisable)
// 	t.save.SetState(tk.StateDisable)
// 	tk.Update()
// 	portals := []lib.Portal{}
// 	base := []int{}
// 	for _, portal := range t.portals {
// 		if !t.disabledPortals[portal.Guid] {
// 			portals = append(portals, portal)
// 			if t.basePortals[portal.Guid] {
// 				base = append(base, len(portals)-1)
// 			}
// 		}
// 	}
// 	t.b0, t.b1, t.solution0, t.solution1 = lib.LargestDoubleHerringbone(portals, base, runtime.GOMAXPROCS(0), func(val int, max int) { t.onProgress(val, max) })
// 	if t.solutionMap != nil {
// 		t.solutionMap.SetSolution([][]lib.Portal{lib.DoubleHerringbonePolyline(t.b0, t.b1, t.solution0, t.solution1)})
// 	}
// 	solutionText := fmt.Sprintf("Solution length: %d + %d", len(t.solution0), len(t.solution1))
// 	t.solutionLabel.SetText(solutionText)
// 	t.add.SetState(tk.StateNormal)
// 	t.reset.SetState(tk.StateNormal)
// 	t.find.SetState(tk.StateNormal)
// 	t.save.SetState(tk.StateNormal)
// 	tk.Update()
// }

// func (t *doubleHerringboneTab) solutionString() string {
// 	return lib.DoubleHerringboneDrawToolsString(t.b0, t.b1, t.solution0, t.solution1)
// }
// func (t *doubleHerringboneTab) EnablePortal(guid string) {
// 	delete(t.disabledPortals, guid)
// 	t.portalStateChanged(guid)
// }
// func (t *doubleHerringboneTab) DisablePortal(guid string) {
// 	t.disabledPortals[guid] = true
// 	delete(t.basePortals, guid)
// 	t.portalStateChanged(guid)
// }
// func (t *doubleHerringboneTab) MakeBase(guid string) {
// 	t.basePortals[guid] = true
// 	t.portalStateChanged(guid)
// }
// func (t *doubleHerringboneTab) UnmakeBase(guid string) {
// 	delete(t.basePortals, guid)
// 	t.portalStateChanged(guid)
// }

// type doubleHerringbonePortalContextMenu struct {
// 	*tk.Menu
// }

// func NewDoubleHerringbonePortalContextMenu(parent tk.Widget, guid string, t *doubleHerringboneTab) *doubleHerringbonePortalContextMenu {
// 	l := &doubleHerringbonePortalContextMenu{}
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
// 	if t.basePortals[guid] {
// 		unbaseAction := tk.NewAction("Unmake base portal")
// 		unbaseAction.OnCommand(func() { t.UnmakeBase(guid) })
// 		l.AddAction(unbaseAction)
// 	} else if !t.disabledPortals[guid] && len(t.basePortals) < 2 {
// 		baseAction := tk.NewAction("Make base portal")
// 		baseAction.OnCommand(func() { t.MakeBase(guid) })
// 		l.AddAction(baseAction)
// 	}
// 	return l
// }
