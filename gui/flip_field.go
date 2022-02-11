package main

import (
	"fmt"
	"image/color"
	"runtime"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/golang/geo/s2"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type flipFieldTab struct {
	*baseTab
	maxFlipPortals           *widget.Entry
	numBackbonePortals       *widget.Entry
	exactBackbonePortalLimit *widget.Check
	simpleBackbone           *widget.Check
	backbone                 []lib.Portal
	rest                     []lib.Portal
	basePortals              map[string]struct{}
}

var _ pattern = (*flipFieldTab)(nil)

func NewFlipFieldTab(app fyne.App, parent fyne.Window, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *container.TabItem {
	t := &flipFieldTab{}
	t.baseTab = NewBaseTab(app, parent, "Flip Field", t, conf, tileFetcher)
	t.basePortals = make(map[string]struct{})
	warningLabel := widget.NewLabel("EXPERIMENTAL: SLOW AND INACCURATE")

	numBackbonePortalsLabel := widget.NewLabel("Num backbone portals: ")
	t.numBackbonePortals = widget.NewEntry()
	t.numBackbonePortals.SetText("16")
	exactBackbonePortalLimitLabel := widget.NewLabel("Exactly: ")
	t.exactBackbonePortalLimit = widget.NewCheck("", func(bool) {})
	simpleBackboneLabel := widget.NewLabel("Simple backbone: ")
	t.simpleBackbone = widget.NewCheck("", func(bool) {})
	maxFlipPortalsLabel := widget.NewLabel("Max flip portals: ")
	t.maxFlipPortals = widget.NewEntry()
	t.maxFlipPortals.SetText("9999")
	content := container.New(
		layout.NewGridLayout(2),
		numBackbonePortalsLabel, t.numBackbonePortals,
		exactBackbonePortalLimitLabel, t.exactBackbonePortalLimit,
		simpleBackboneLabel, t.simpleBackbone,
		maxFlipPortalsLabel, t.maxFlipPortals)
	topContent := container.NewVBox(
		warningLabel,
		container.NewHBox(t.add, t.reset),
		content,
		container.NewHBox(t.find, t.save, t.copy, t.solutionLabel),
		t.progress)
	return container.NewTabItem("Flip Field",
		container.New(
			layout.NewBorderLayout(topContent, nil, nil, nil),
			topContent))
}

func (t *flipFieldTab) onReset() {
	t.basePortals = make(map[string]struct{})
}

func (t *flipFieldTab) portalLabel(guid string) string {
	if _, ok := t.disabledPortals[guid]; ok {
		return "Disabled"
	}
	if _, ok := t.basePortals[guid]; ok {
		return "Base"
	}
	return "Normal"
}

func (t *flipFieldTab) portalColor(guid string) color.NRGBA {
	if _, ok := t.disabledPortals[guid]; ok {
		if _, ok := t.selectedPortals[guid]; !ok {
			return color.NRGBA{128, 128, 128, 255}
		}
		return color.NRGBA{64, 64, 64, 255}
	}
	if _, ok := t.selectedPortals[guid]; !ok {
		return color.NRGBA{255, 170, 0, 255}
	}
	return color.NRGBA{255, 0, 0, 255}
}

func (t *flipFieldTab) onContextMenu(x, y float32) {
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

func (t *flipFieldTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			continue
		}
		t.disabledPortals[guid] = struct{}{}
		delete(t.basePortals, guid)
		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}
func (t *flipFieldTab) makeSelectedPortalsBases() {
	for guid := range t.selectedPortals {
		if _, ok := t.basePortals[guid]; ok {
			continue
		}
		t.basePortals[guid] = struct{}{}
		delete(t.disabledPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
func (t *flipFieldTab) unmakeSelectedPortalsBases() {
	for guid := range t.selectedPortals {
		if _, ok := t.basePortals[guid]; !ok {
			continue
		}
		delete(t.basePortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}

func (t *flipFieldTab) search() {
	if len(t.portals) < 3 {
		return
	}

	portals := []lib.Portal{}
	base := []int{}
	for i, portal := range t.portals {
		if _, ok := t.disabledPortals[portal.Guid]; !ok {
			portals = append(portals, portal)
			if _, ok := t.basePortals[portal.Guid]; ok {
				base = append(base, i)
			}
		}
	}
	maxFlipPortals, err := strconv.Atoi(t.maxFlipPortals.Text)
	if err != nil || maxFlipPortals < 1 {
		return
	}
	numBackbonePortals, err := strconv.Atoi(t.numBackbonePortals.Text)
	if err != nil || numBackbonePortals < 1 {
		return
	}
	backbonePortalLimit := lib.FlipFieldBackbonePortalLimit{Value: numBackbonePortals}
	if t.exactBackbonePortalLimit.Checked {
		backbonePortalLimit.LimitType = lib.EQUAL
	} else {
		backbonePortalLimit.LimitType = lib.LESS_EQUAL
	}
	options := []lib.FlipFieldOption{
		lib.FlipFieldProgressFunc(
			func(val int, max int) { t.onProgress(val, max) }),
		backbonePortalLimit,
		lib.FlipFieldFixedBaseIndices(base),
		lib.FlipFieldSimpleBackbone(t.simpleBackbone.Checked),
		lib.FlipFieldMaxFlipPortals(maxFlipPortals),
		lib.FlipFieldNumWorkers(runtime.GOMAXPROCS(0)),
	}
	t.add.Disable()
	t.reset.Disable()
	t.maxFlipPortals.Disable()
	t.numBackbonePortals.Disable()
	t.exactBackbonePortalLimit.Disable()
	t.simpleBackbone.Disable()
	t.find.Disable()
	t.save.Disable()
	t.copy.Disable()
	t.backbone, t.rest = lib.LargestFlipField(portals, options...)
	hull := s2.NewConvexHullQuery()
	for _, p := range t.rest {
		hull.AddPoint(s2.PointFromLatLng(p.LatLng))
	}
	hullPoints := hull.ConvexHull().Vertices()
	if len(hullPoints) > 0 {
		hullPoints = append(hullPoints, hullPoints[0])
	}

	if t.solutionMap != nil {
		t.solutionMap.setSolutionPoints([][]s2.Point{portalsToPoints(t.backbone), hullPoints})
	}

	solutionText := fmt.Sprintf("Num backbone portals: %d, num flip portals: %d", len(t.backbone), len(t.rest))
	t.solutionLabel.SetText(solutionText)
	t.add.Enable()
	t.reset.Enable()
	t.maxFlipPortals.Enable()
	t.numBackbonePortals.Enable()
	t.exactBackbonePortalLimit.Enable()
	t.simpleBackbone.Enable()
	t.find.Enable()
	t.save.Enable()
	t.copy.Enable()
}

func (t *flipFieldTab) solutionString() string {
	s := fmt.Sprintf("[%s", lib.PolylineFromPortalList(t.backbone))
	if len(t.rest) > 0 {
		s += fmt.Sprintf(",%s", lib.MarkersFromPortalList(t.rest))
	}
	return s + "]"
}
