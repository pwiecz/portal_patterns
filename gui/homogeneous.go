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
	anchorPortals map[string]struct{}
}

var _ pattern = (*homogeneousTab)(nil)

func NewHomogeneousTab(app fyne.App, parent fyne.Window, conf *configuration.Configuration, tileFetcher *osm.MapTiles) *container.TabItem {
	t := &homogeneousTab{}
	t.baseTab = NewBaseTab(app, parent, "Homogeneous", conf, tileFetcher)
	t.pattern = t
	t.anchorPortals = make(map[string]struct{})
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
			topContent, t.portalList))
}

func (t *homogeneousTab) onReset() {
	t.anchorPortals = make(map[string]struct{})
}

func (t *homogeneousTab) portalLabel(guid string) string {
	if _, ok := t.disabledPortals[guid]; ok {
		return "Disabled"
	}
	if _, ok := t.anchorPortals[guid]; ok {
		return "Anchor"
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
	if _, ok := t.anchorPortals[guid]; ok {
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
	menuItems := []*fyne.MenuItem{
		fyne.NewMenuItem("Disable portals", t.disableSelectedPortals),
		fyne.NewMenuItem("Enable portals", t.enableSelectedPortals),
		fyne.NewMenuItem("Make anchors", t.makeSelectedPortalsAnchors),
		fyne.NewMenuItem("Unmake anchors", t.unmakeSelectedPortalsAnchors)}
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
	anchors := []int{}
	for _, portal := range t.portals {
		if _, ok := t.disabledPortals[portal.Guid]; !ok {
			portals = append(portals, portal)
			if _, ok := t.anchorPortals[portal.Guid]; ok {
				anchors = append(anchors, len(portals)-1)
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
	t.add.Disable()
	t.reset.Disable()
	t.maxDepth.Disable()
	t.innerPortals.Disable()
	t.pure.Disable()
	t.topTriangle.Disable()
	t.find.Disable()
	t.save.Disable()
	t.copy.Disable()
	options = append(options, lib.HomogeneousFixedCornerIndices(anchors))

	t.solutionLabel.SetText("")
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

// func (t *homogeneousTab) EnablePortal(guid string) {
// 	delete(t.disabledPortals, guid)
// 	t.portalStateChanged(guid)
// }
// func (t *homogeneousTab) DisablePortal(guid string) {
// 	t.disabledPortals[guid] = true
// 	delete(t.anchorPortals, guid)
// 	t.portalStateChanged(guid)
// }
func (t *homogeneousTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			continue
		}
		t.disabledPortals[guid] = struct{}{}
		delete(t.anchorPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.pattern.portalColor(guid))
	}
}
func (t *homogeneousTab) makeSelectedPortalsAnchors() {
	for guid := range t.selectedPortals {
		if _, ok := t.anchorPortals[guid]; ok {
			continue
		}
		t.anchorPortals[guid] = struct{}{}
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}
func (t *homogeneousTab) unmakeSelectedPortalsAnchors() {
	for guid := range t.selectedPortals {
		if _, ok := t.anchorPortals[guid]; !ok {
			continue
		}
		delete(t.anchorPortals, guid)
		t.solutionMap.SetPortalColor(guid, t.portalColor(guid))
	}
}

// func (t *homogeneousTab) UnmakeAnchor(guid string) {
// 	delete(t.anchorPortals, guid)
// 	t.portalStateChanged(guid)
// }

// type homogeneousPortalContextMenu struct {
// 	*tk.Menu
// }

// func NewHomogeneousPortalContextMenu(parent tk.Widget, guid string, t *homogeneousTab) *homogeneousPortalContextMenu {
// 	l := &homogeneousPortalContextMenu{}
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
// 	if t.anchorPortals[guid] {
// 		unanchorAction := tk.NewAction("Unmake anchor")
// 		unanchorAction.OnCommand(func() { t.UnmakeAnchor(guid) })
// 		l.AddAction(unanchorAction)
// 	} else if !t.disabledPortals[guid] && len(t.anchorPortals) < 3 {
// 		anchorAction := tk.NewAction("Make anchor")
// 		anchorAction.OnCommand(func() { t.MakeAnchor(guid) })
// 		l.AddAction(anchorAction)
// 	}
// 	return l
// }
