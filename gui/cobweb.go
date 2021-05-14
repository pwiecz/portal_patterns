package main

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type cobwebTab struct {
	*baseTab
	solution      []lib.Portal
	cornerPortals map[string]struct{}
}

var _ pattern = (*cobwebTab)(nil)

func NewCobwebTab(app fyne.App, parent fyne.Window, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *container.TabItem {
	t := &cobwebTab{}
	t.baseTab = NewBaseTab(app, parent, "Cobweb", t, conf, tileFetcher)
	t.cornerPortals = make(map[string]struct{})
	content := container.New(
		layout.NewGridLayout(2))
	topContent := container.NewVBox(
		container.NewHBox(t.add, t.reset),
		content,
		container.NewHBox(t.find, t.save, t.copy, t.solutionLabel),
		t.progress,
	)
	return container.NewTabItem("Cobweb",
		container.New(
			layout.NewBorderLayout(topContent, nil, nil, nil),
			topContent))
}

func (t *cobwebTab) onReset() {
	t.cornerPortals = make(map[string]struct{})
}

func (t *cobwebTab) portalLabel(guid string) string {
	if _, ok := t.disabledPortals[guid]; ok {
		return "Disabled"
	}
	if _, ok := t.cornerPortals[guid]; ok {
		return "Corner"
	}
	return "Normal"
}

func (t *cobwebTab) portalColor(guid string) color.NRGBA {
	if _, ok := t.disabledPortals[guid]; ok {
		if _, ok := t.selectedPortals[guid]; !ok {
			return color.NRGBA{128, 128, 128, 255}
		}
		return color.NRGBA{64, 64, 64, 255}
	}
	if _, ok := t.cornerPortals[guid]; ok {
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

func (t *cobwebTab) onContextMenu(x, y float32) {
	menuItems := []*fyne.MenuItem{}
	var isDisabledSelected, isEnabledSelected, isCornerSelected bool
	numNonCornerSelected := 0
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			isDisabledSelected = true
		} else {
			isEnabledSelected = true
		}
		if _, ok := t.cornerPortals[guid]; ok {
			isCornerSelected = true
		} else {
			numNonCornerSelected++
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
	if numNonCornerSelected > 0 && numNonCornerSelected+len(t.cornerPortals) <= 3 {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Make corner", t.makeSelectedPortalsCorners))
	}
	if isCornerSelected {
		menuItems = append(menuItems,
			fyne.NewMenuItem("Unmake corner", t.unmakeSelectedPortalsCorners))
	}
	if len(menuItems) == 0 {
		return
	}
	menu := fyne.NewMenu("", menuItems...)
	menu.Items = menuItems
	widget.ShowPopUpMenuAtPosition(menu, t.app.Driver().CanvasForObject(t.solutionMap),
		fyne.NewPos(x, y))
}

func (t *cobwebTab) search() {
	if len(t.portals) < 3 {
		return
	}
	portals := []lib.Portal{}
	corner := []int{}
	for _, portal := range t.portals {
		if _, ok := t.disabledPortals[portal.Guid]; !ok {
			portals = append(portals, portal)
			if _, ok := t.cornerPortals[portal.Guid]; ok {
				corner = append(corner, len(portals)-1)
			}
		}
	}

	t.add.Disable()
	t.reset.Disable()
	t.find.Disable()
	t.save.Disable()
	t.copy.Disable()

	t.solutionLabel.SetText("")
	t.solution = lib.LargestCobweb(portals, corner, func(val int, max int) { t.onProgress(val, max) })

	if t.solutionMap != nil {
		t.solutionMap.SetSolution([][]lib.Portal{lib.CobwebPolyline(t.solution)})
	}
	solutionText := fmt.Sprintf("Solution length: %d", len(t.solution))
	t.solutionLabel.SetText(solutionText)

	t.add.Enable()
	t.reset.Enable()
	t.find.Enable()
	t.save.Enable()
	t.copy.Enable()
}

func (t *cobwebTab) solutionString() string {
	return lib.CobwebDrawToolsString(t.solution)
}

func (t *cobwebTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			continue
		}
		t.disabledPortals[guid] = struct{}{}
		delete(t.cornerPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}
func (t *cobwebTab) makeSelectedPortalsCorners() {
	for guid := range t.selectedPortals {
		if _, ok := t.cornerPortals[guid]; ok {
			continue
		}
		t.cornerPortals[guid] = struct{}{}
		delete(t.cornerPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
func (t *cobwebTab) unmakeSelectedPortalsCorners() {
	for guid := range t.selectedPortals {
		if _, ok := t.cornerPortals[guid]; !ok {
			continue
		}
		delete(t.cornerPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
