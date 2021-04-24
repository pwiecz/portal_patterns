package main

// import "fmt"
// import "runtime"

// import "github.com/pwiecz/portal_patterns/gui/osm"
// import "github.com/pwiecz/portal_patterns/configuration"
// import "github.com/pwiecz/portal_patterns/lib"
// import "github.com/pwiecz/atk/tk"

// type herringboneTab struct {
// 	*baseTab
// 	b0, b1      lib.Portal
// 	solution    []lib.Portal
// 	basePortals map[string]bool
// }

// func NewHerringboneTab(parent tk.Widget, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *herringboneTab {
// 	t := &herringboneTab{}
// 	t.baseTab = NewBaseTab(parent, "Herringbone", conf, tileFetcher)
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

// func (t *herringboneTab) onReset() {
// 	t.basePortals = make(map[string]bool)
// }

// func (t *herringboneTab) portalLabel(guid string) string {
// 	if t.disabledPortals[guid] {
// 		return "Disabled"
// 	}
// 	if t.basePortals[guid] {
// 		return "Base"
// 	}
// 	return "Normal"
// }

// func (t *herringboneTab) portalColor(guid string) string {
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

// func (t *herringboneTab) onPortalContextMenu(guid string, x, y int) {
// 	menu := NewHerringbonePortalContextMenu(tk.RootWindow(), guid, t)
// 	tk.PopupMenu(menu.Menu, x, y)
// }

// func (t *herringboneTab) search() {
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
// 	t.b0, t.b1, t.solution = lib.LargestHerringbone(portals, base, runtime.GOMAXPROCS(0), func(val int, max int) { t.onProgress(val, max) })
// 	if t.solutionMap != nil {
// 		t.solutionMap.SetSolution([][]lib.Portal{lib.HerringbonePolyline(t.b0, t.b1, t.solution)})
// 	}
// 	solutionText := fmt.Sprintf("Solution length: %d", len(t.solution))
// 	t.solutionLabel.SetText(solutionText)
// 	t.add.SetState(tk.StateNormal)
// 	t.reset.SetState(tk.StateNormal)
// 	t.find.SetState(tk.StateNormal)
// 	t.save.SetState(tk.StateNormal)
// 	tk.Update()
// }

// func (t *herringboneTab) solutionString() string {
// 	return lib.HerringboneDrawToolsString(t.b0, t.b1, t.solution)
// }
// func (t *herringboneTab) EnablePortal(guid string) {
// 	delete(t.disabledPortals, guid)
// 	t.portalStateChanged(guid)
// }
// func (t *herringboneTab) DisablePortal(guid string) {
// 	t.disabledPortals[guid] = true
// 	delete(t.basePortals, guid)
// 	t.portalStateChanged(guid)
// }
// func (t *herringboneTab) MakeBase(guid string) {
// 	t.basePortals[guid] = true
// 	t.portalStateChanged(guid)
// }
// func (t *herringboneTab) UnmakeBase(guid string) {
// 	delete(t.basePortals, guid)
// 	t.portalStateChanged(guid)
// }

// type herringbonePortalContextMenu struct {
// 	*tk.Menu
// }

// func NewHerringbonePortalContextMenu(parent tk.Widget, guid string, t *herringboneTab) *herringbonePortalContextMenu {
// 	l := &herringbonePortalContextMenu{}
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
