package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"runtime"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type homogeneousTab struct {
	*baseTab
	maxDepth      *widget.Entry
	innerPortals  *widget.Select
	pure          *widget.Check
	topTriangle   *widget.Select
	solution      []lib.Portal
	depth         uint16
	cornerPortals map[string]struct{}
}

var _ pattern = (*homogeneousTab)(nil)

func NewHomogeneousTab(app fyne.App, parent fyne.Window, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *container.TabItem {
	t := &homogeneousTab{}
	t.baseTab = NewBaseTab(app, parent, "Homogeneous", t, conf, tileFetcher)
	t.cornerPortals = make(map[string]struct{})
	maxDepthLabel := widget.NewLabel("Max depth: ")
	t.maxDepth = widget.NewEntry()
	t.maxDepth.SetText("6")
	innerPortalsLabel := widget.NewLabel("Inner portal positions: ")
	t.innerPortals = widget.NewSelect([]string{"Arbitrary", "Spread around (slow)"}, func(string) {})
	t.innerPortals.SetSelectedIndex(0)
	topTriangleLabel := widget.NewLabel("Top triangle: ")
	t.topTriangle = widget.NewSelect([]string{"Smallest Area", "Largest Area", "Most Equilateral", "Random"}, func(string) {})
	t.topTriangle.SetSelectedIndex(0)
	pureLabel := widget.NewLabel("Pure: ")
	t.pure = widget.NewCheck("", func(bool) {})
	content := container.New(
		layout.NewGridLayout(2),
		maxDepthLabel, t.maxDepth,
		innerPortalsLabel, t.innerPortals,
		topTriangleLabel, t.topTriangle,
		pureLabel, t.pure)
	topContent := container.NewVBox(
		container.NewHBox(t.add, t.reset),
		content,
		container.NewHBox(t.find, t.save, t.copy, t.solutionLabel),
		t.progress,
	)
	return container.NewTabItem("Homogeneous",
		container.New(
			layout.NewBorderLayout(topContent, nil, nil, nil),
			topContent))
}

func (t *homogeneousTab) onReset() {
	t.cornerPortals = make(map[string]struct{})
}

func (t *homogeneousTab) portalLabel(guid string) string {
	if _, ok := t.disabledPortals[guid]; ok {
		return "Disabled"
	}
	if _, ok := t.cornerPortals[guid]; ok {
		return "Corner"
	}
	return "Normal"
}

func (t *homogeneousTab) portalColor(guid string) color.NRGBA {
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

func (t *homogeneousTab) onContextMenu(x, y float32) {
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

func (t *homogeneousTab) search() {
	if len(t.portals) < 3 {
		return
	}
	portals := []lib.Portal{}
	corners := []int{}
	for _, portal := range t.portals {
		if _, ok := t.disabledPortals[portal.Guid]; !ok {
			portals = append(portals, portal)
			if _, ok := t.cornerPortals[portal.Guid]; ok {
				corners = append(corners, len(portals)-1)
			}
		}
	}

	options := []lib.HomogeneousOption{lib.HomogeneousNumWorkers(runtime.GOMAXPROCS(0))}
	maxDepth, err := strconv.Atoi(t.maxDepth.Text)
	if err != nil || maxDepth < 1 {
		return
	}
	options = append(options, lib.HomogeneousMaxDepth(maxDepth))
	options = append(options, lib.HomogeneousPure(t.pure.Checked))
	// set inner portals opion before setting top level scorer, as inner scorer
	// overwrites the top level scorer
	if t.innerPortals.SelectedIndex() == 1 {
		options = append(options, lib.HomogeneousSpreadAround{})
	}
	if t.topTriangle.SelectedIndex() == 0 {
		options = append(options, lib.HomogeneousSmallestArea{})
	} else if t.topTriangle.SelectedIndex() == 1 {
		options = append(options, lib.HomogeneousLargestArea{})
	} else if t.topTriangle.SelectedIndex() == 2 {
		options = append(options, lib.HomogeneousMostEquilateralTriangle{})
	} else if t.topTriangle.SelectedIndex() == 3 {
		rand := rand.New(rand.NewSource(time.Now().UnixNano()))
		options = append(options, lib.HomogeneousRandom{Rand: rand})
	}
	options = append(options, lib.HomogeneousProgressFunc(t.onProgress))
	options = append(options, lib.HomogeneousFixedCornerIndices(corners))
	disabledPortals := t.getDisabledPortals()
	if len(disabledPortals) > 0 {
		options = append(options, lib.HomogeneousDisabledPortals(disabledPortals))
	}

	t.solutionLabel.SetText("")

	t.add.Disable()
	t.reset.Disable()
	t.maxDepth.Disable()
	t.innerPortals.Disable()
	t.pure.Disable()
	t.topTriangle.Disable()
	t.find.Disable()
	t.save.Disable()
	t.copy.Disable()
	t.solution, t.depth = lib.DeepestHomogeneous(portals, options...)

	if t.solutionMap != nil {
		t.solutionMap.SetSolution(lib.HomogeneousPolylines(t.depth, t.solution))
	}
	var solutionText string
	if t.depth > 0 {
		solutionText = fmt.Sprintf("Solution depth: %d", t.depth)
	} else {
		solutionText = "No solution found"
	}
	t.solutionLabel.SetText(solutionText)
	t.add.Enable()
	t.reset.Enable()
	t.maxDepth.Enable()
	t.innerPortals.Enable()
	t.pure.Enable()
	t.topTriangle.Enable()
	t.find.Enable()
	t.save.Enable()
	t.copy.Enable()
}

func (t *homogeneousTab) solutionString() string {
	return lib.HomogeneousDrawToolsString(t.depth, t.solution)
}

func (t *homogeneousTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			continue
		}
		t.disabledPortals[guid] = struct{}{}
		delete(t.cornerPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}
func (t *homogeneousTab) makeSelectedPortalsCorners() {
	for guid := range t.selectedPortals {
		if _, ok := t.cornerPortals[guid]; ok {
			continue
		}
		t.cornerPortals[guid] = struct{}{}
		delete(t.disabledPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
func (t *homogeneousTab) unmakeSelectedPortalsCorners() {
	for guid := range t.selectedPortals {
		if _, ok := t.cornerPortals[guid]; !ok {
			continue
		}
		delete(t.cornerPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
