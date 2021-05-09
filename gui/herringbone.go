package main

import (
	"fmt"
	"image/color"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type herringboneTab struct {
	*baseTab
	b0, b1      lib.Portal
	solution    []lib.Portal
	basePortals map[string]struct{}
}

var _ pattern = (*herringboneTab)(nil)

func NewHerringboneTab(app fyne.App, parent fyne.Window, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *container.TabItem {
	t := &herringboneTab{}
	t.baseTab = NewBaseTab(app, parent, "Herringbone", conf, tileFetcher)
	t.pattern = t
	content := container.New(
		layout.NewGridLayout(2))
	topContent := container.NewVBox(
		container.NewHBox(t.add, t.reset),
		content,
		container.NewHBox(t.find, t.save, t.copy, t.solutionLabel),
		t.progress,
	)
	return container.NewTabItem("Herringbone",
		container.New(
			layout.NewBorderLayout(topContent, nil, nil, nil),
			topContent, t.portalList))
}

func (t *herringboneTab) onReset() {
	t.basePortals = make(map[string]struct{})
}

func (t *herringboneTab) portalLabel(guid string) string {
	if _, ok := t.disabledPortals[guid]; ok {
		return "Disabled"
	}
	if _, ok := t.basePortals[guid]; ok {
		return "Base"
	}
	return "Normal"
}

func (t *herringboneTab) portalColor(guid string) color.NRGBA {
	if _, ok := t.disabledPortals[guid]; ok {
		if _, ok := t.selectedPortals[guid]; !ok {
			return color.NRGBA{128, 128, 128, 255}
		}
		return color.NRGBA{64, 64, 64, 255}
	}
	if _, ok := t.basePortals[guid]; ok {
		if _, ok := t.selectedPortals[guid]; !ok {
			return color.NRGBA{0, 255, 0, 255}
		}
		return color.NRGBA{0, 128, 0, 255}
	}
	if _, ok := t.selectedPortals[guid]; !ok {
		return color.NRGBA{255, 170, 0, 255}
	}
	return color.NRGBA{255, 0, 0, 255}
}

func (t *herringboneTab) onContextMenu(x, y float32) {
	menuItems := []*fyne.MenuItem{
		fyne.NewMenuItem("Disable portals", t.disableSelectedPortals),
		fyne.NewMenuItem("Enable portals", t.enableSelectedPortals),
		fyne.NewMenuItem("Make base", t.makeSelectedPortalsBase),
		fyne.NewMenuItem("Unmake base", t.unmakeSelectedPortalsBase)}
	menu := fyne.NewMenu("", menuItems...)
	menu.Items = menuItems
	widget.ShowPopUpMenuAtPosition(menu, t.app.Driver().CanvasForObject(t.solutionMap),
		fyne.NewPos(x, y))
	// 	menu := NewHerringbonePortalContextMenu(tk.RootWindow(), guid, t)
	// 	tk.PopupMenu(menu.Menu, x, y)
}

func (t *herringboneTab) search() {
	if len(t.portals) < 3 {
		return
	}
	portals := []lib.Portal{}
	base := []int{}
	for _, portal := range t.portals {
		if _, ok := t.disabledPortals[portal.Guid]; !ok {
			portals = append(portals, portal)
			if _, ok := t.basePortals[portal.Guid]; ok {
				base = append(base, len(portals)-1)
			}
		}
	}

	t.add.Disable()
	t.reset.Disable()
	t.find.Disable()
	t.save.Disable()
	t.copy.Disable()

	t.solutionLabel.SetText("")
	t.b0, t.b1, t.solution = lib.LargestHerringbone(portals, base, runtime.GOMAXPROCS(0), func(val int, max int) { t.onProgress(val, max) })

	if t.solutionMap != nil {
		t.solutionMap.SetSolution([][]lib.Portal{lib.HerringbonePolyline(t.b0, t.b1, t.solution)})
	}
	solutionText := fmt.Sprintf("Solution length: %d", len(t.solution))
	t.solutionLabel.SetText(solutionText)

	t.add.Enable()
	t.reset.Enable()
	t.find.Enable()
	t.save.Enable()
	t.copy.Enable()
}

func (t *herringboneTab) solutionString() string {
	return lib.HerringboneDrawToolsString(t.b0, t.b1, t.solution)
}

// func (t *herringboneTab) EnablePortal(guid string) {
// 	delete(t.disabledPortals, guid)
// 	t.portalStateChanged(guid)
// }
// func (t *herringboneTab) DisablePortal(guid string) {
// 	t.disabledPortals[guid] = true
// 	delete(t.anchorPortals, guid)
// 	t.portalStateChanged(guid)
// }
func (t *herringboneTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			continue
		}
		t.disabledPortals[guid] = struct{}{}
		delete(t.basePortals, guid)
		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}
func (t *herringboneTab) makeSelectedPortalsBase() {
	for guid := range t.selectedPortals {
		if _, ok := t.basePortals[guid]; ok {
			continue
		}
		t.basePortals[guid] = struct{}{}
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
func (t *herringboneTab) unmakeSelectedPortalsBase() {
	for guid := range t.selectedPortals {
		if _, ok := t.basePortals[guid]; !ok {
			continue
		}
		delete(t.basePortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}

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
