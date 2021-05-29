package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type homogeneousTab struct {
	*baseTab
	maxDepth      *fltk.Spinner
	innerPortals  *fltk.Choice
	topLevel      *fltk.Choice
	pure          *fltk.CheckButton
	depth         uint16
	solution      []lib.Portal
	anchorPortals map[string]struct{}
}

func newHomogeneousTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *homogeneousTab {
	t := &homogeneousTab{
		anchorPortals: make(map[string]struct{}),
	}
	t.baseTab = newBaseTab("Homogeneous", configuration, tileFetcher, t)

	maxDepthPack := fltk.NewPack(0, 0, 760, 30)
	maxDepthPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.maxDepth = fltk.NewSpinner(0, 0, 200, 30, "Max depth:")
	t.maxDepth.SetMinimum(1)
	t.maxDepth.SetMaximum(8)
	t.maxDepth.SetValue(6)
	t.maxDepth.SetType(fltk.SPINNER_INT_INPUT)
	maxDepthPack.End()
	t.Add(maxDepthPack)

	innerPortalsPack := fltk.NewPack(0, 0, 760, 30)
	innerPortalsPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.innerPortals = fltk.NewChoice(0, 0, 200, 30, "Inner portal positions:")
	t.innerPortals.Add("Arbitrary", func() {})
	t.innerPortals.Add("Spread around (slow)", func() {})
	t.innerPortals.SetValue(0)
	innerPortalsPack.End()
	t.Add(innerPortalsPack)

	topLevelPack := fltk.NewPack(0, 0, 760, 30)
	topLevelPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.topLevel = fltk.NewChoice(0, 0, 200, 30, "Top level triangle:")
	t.topLevel.Add("Smallest area", func() {})
	t.topLevel.Add("Largest area", func() {})
	t.topLevel.Add("Most Equilateral", func() {})
	t.topLevel.Add("Random", func() {})
	t.topLevel.SetValue(0)
	topLevelPack.End()
	t.Add(topLevelPack)

	purePack := fltk.NewPack(0, 0, 760, 30)
	purePack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.pure = fltk.NewCheckButton(0, 0, 200, 30, "Pure")
	purePack.End()
	t.Add(purePack)

	t.Add(t.searchSaveCopyPack)
	t.Add(t.progress)
	if t.portalList != nil {
		t.Add(t.portalList)
	}
	t.End()

	return t
}

func (t *homogeneousTab) onReset() {
	t.anchorPortals = make(map[string]struct{})
	t.depth = 0
	t.solution = nil
}
func (t *homogeneousTab) onSearch() {
	options := []lib.HomogeneousOption{
		lib.HomogeneousMaxDepth(t.maxDepth.Value()),
		lib.HomogeneousProgressFunc(func(val, max int) {
			fltk.Awake(func() {
				t.progress.SetMaximum(float64(max))
				t.progress.SetValue(float64(val))
			})
		}),
	}
	if t.pure.Value() {
		options = append(options, lib.HomogeneousPure(true))
	}
	if t.innerPortals.Value() == 1 {
		options = append(options, lib.HomogeneousSpreadAround{})
	}
	switch t.topLevel.Value() {
	case 0:
		options = append(options, lib.HomogeneousSmallestArea{})
	case 1:
		options = append(options, lib.HomogeneousLargestArea{})
	case 2:
		options = append(options, lib.HomogeneousMostEquilateralTriangle{})
	case 3:
		rand := rand.New(rand.NewSource(time.Now().UnixNano()))
		options = append(options, lib.HomogeneousRandom{Rand: rand})
	}
	go func() {
		portals := t.enabledPortals()
		anchors := []int{}
		for i, portal := range portals {
			if _, ok := t.anchorPortals[portal.Guid]; ok {
				anchors = append(anchors, i)
			}
		}
		options = append(options, lib.HomogeneousFixedCornerIndices(anchors))
		t.solution, t.depth = lib.DeepestHomogeneous(portals, options...)
		if t.mapWindow != nil {
			t.mapWindow.SetPortalPaths(lib.HomogeneousPolylines(t.depth, t.solution))
		}
		fltk.Awake(func() {
			var solutionText string
			if t.depth > 0 {
				solutionText = fmt.Sprintf("Solution depth: %d", t.depth)
			} else {
				solutionText = "No solution found"
			}
			t.onSearchDone(solutionText)
		})
	}()
}

func (t *homogeneousTab) solutionString() string {
	return lib.HomogeneousDrawToolsString(t.depth, t.solution)
}

func (t *homogeneousTab) portalLabel(guid string) string {
	if _, ok := t.anchorPortals[guid]; ok {
		return "Anchor"
	}
	return t.baseTab.portalLabel(guid)
}
func (t *homogeneousTab) portalColor(guid string) color.Color {
	if _, ok := t.anchorPortals[guid]; ok {
		if _, ok := t.selectedPortals[guid]; ok {
			return color.NRGBA{0, 255, 0, 128}
		}
		return color.NRGBA{0, 128, 0, 128}
	}
	return t.baseTab.portalColor(guid)
}

func (t *homogeneousTab) enableSelectedPortals() {
	for guid := range t.selectedPortals {
		delete(t.disabledPortals, guid)
		if t.mapWindow != nil {
			t.mapWindow.SetPortalColor(guid, t.pattern.portalColor(guid))
		}
		if t.portalList != nil {
			t.portalList.SetPortalLabel(guid, t.pattern.portalLabel(guid))
		}
	}
	if t.portalList != nil {
		t.portalList.Redraw()
	}
}

func (t *homogeneousTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		t.disabledPortals[guid] = struct{}{}
		delete(t.anchorPortals, guid)
		if t.mapWindow != nil {
			t.mapWindow.SetPortalColor(guid, t.pattern.portalColor(guid))
			t.mapWindow.Lower(guid)
		}
		if t.portalList != nil {
			t.portalList.SetPortalLabel(guid, t.pattern.portalLabel(guid))
		}
	}
	if t.portalList != nil {
		t.portalList.Redraw()
	}
}

func (t *homogeneousTab) makeSelectedPortalsAnchors() {
	for guid := range t.selectedPortals {
		delete(t.disabledPortals, guid)
		t.anchorPortals[guid] = struct{}{}
		if t.mapWindow != nil {
			t.mapWindow.SetPortalColor(guid, t.portalColor(guid))
			t.mapWindow.Raise(guid)
		}
		if t.portalList != nil {
			t.portalList.SetPortalLabel(guid, t.portalLabel(guid))
		}
	}
	if t.portalList != nil {
		t.portalList.Redraw()
	}
}
func (t *homogeneousTab) unmakeSelectedPortalsAnchors() {
	for guid := range t.selectedPortals {
		delete(t.anchorPortals, guid)
		if t.mapWindow != nil {
			t.mapWindow.SetPortalColor(guid, t.portalColor(guid))
			t.mapWindow.Raise(guid)
		}
		if t.portalList != nil {
			t.portalList.SetPortalLabel(guid, t.portalLabel(guid))
		}
	}
	if t.portalList != nil {
		t.portalList.Redraw()
	}
}

func (t *homogeneousTab) contextMenu() *menu {
	var aSelectedGUID string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	numSelectedAnchor := 0
	numSelectedNotAnchor := 0
	for guid := range t.selectedPortals {
		aSelectedGUID = guid
		if _, ok := t.disabledPortals[guid]; ok {
			numSelectedDisabled++
		} else {
			numSelectedEnabled++
		}
		if _, ok := t.anchorPortals[guid]; ok {
			numSelectedAnchor++
		} else {
			numSelectedNotAnchor++
		}
	}
	menu := &menu{}
	if len(t.selectedPortals) > 1 {
		menu.header = fmt.Sprintf("%d portals selected", len(t.selectedPortals))
	} else if len(t.selectedPortals) == 1 {
		menu.header = t.portalMap[aSelectedGUID].Name
	}
	if numSelectedDisabled > 0 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Enable", t.enableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Enable All", t.enableSelectedPortals})
		}
	}
	if numSelectedEnabled > 0 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Disable", t.disableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Disable All", t.disableSelectedPortals})
		}
	}
	if numSelectedAnchor > 0 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Unmake anchor", t.unmakeSelectedPortalsAnchors})
		} else {
			menu.items = append(menu.items, menuItem{"Unmake all anchors", t.unmakeSelectedPortalsAnchors})
		}
	}
	if numSelectedNotAnchor > 0 && numSelectedNotAnchor+len(t.anchorPortals) <= 3 {
		if len(t.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Make anchor", t.makeSelectedPortalsAnchors})
		} else {
			menu.items = append(menu.items, menuItem{"Make all anchors", t.makeSelectedPortalsAnchors})
		}
	}
	return menu
}
