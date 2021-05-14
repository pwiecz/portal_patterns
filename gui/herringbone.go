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
	t.baseTab = NewBaseTab(app, parent, "Herringbone", t, conf, tileFetcher)
	t.basePortals = make(map[string]struct{})
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
			topContent))
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
	menuItems := []*fyne.MenuItem{}
	var isDisabledSelected, isEnabledSelected, isBaseSelected bool
	numNonBaseSelected := 0
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			isDisabledSelected = true
		} else {
			isEnabledSelected = true
		}
		if _, ok := t.basePortals[guid]; ok {
			isBaseSelected = true
		} else {
			numNonBaseSelected++
		}
	}
	if isDisabledSelected {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Enable", t.enableSelectedPortals))
	}
	if isEnabledSelected {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Disable", t.disableSelectedPortals))
	}
	if numNonBaseSelected > 0 && numNonBaseSelected+len(t.basePortals) <= 2 {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Make base", t.makeSelectedPortalsBases))
	}
	if isBaseSelected {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Unmake base", t.unmakeSelectedPortalsBases))
	}
	if len(menuItems) == 0 {
		return
	}
	menu := fyne.NewMenu("", menuItems...)
	menu.Items = menuItems
	widget.ShowPopUpMenuAtPosition(menu, t.app.Driver().CanvasForObject(t.solutionMap),
		fyne.NewPos(x, y))
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
func (t *herringboneTab) makeSelectedPortalsBases() {
	for guid := range t.selectedPortals {
		if _, ok := t.basePortals[guid]; ok {
			continue
		}
		t.basePortals[guid] = struct{}{}
		delete(t.disabledPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
func (t *herringboneTab) unmakeSelectedPortalsBases() {
	for guid := range t.selectedPortals {
		if _, ok := t.basePortals[guid]; !ok {
			continue
		}
		delete(t.basePortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
