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

func NewHomogeneousTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *homogeneousTab {
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

func (t *homogeneousTab) onReset() {}
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
			t.mapWindow.SetPaths(lib.HomogeneousPolylines(t.depth, t.solution))
		}
		fltk.Awake(func() {
			var solutionText string
			if t.depth > 0 {
				solutionText = fmt.Sprintf("Solution depth: %d", t.depth)
			} else {
				solutionText = fmt.Sprintf("No solution found")
			}
			t.onSearchDone(solutionText)
		})
	}()
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
		} else {
			return color.NRGBA{0, 128, 0, 128}
		}
	}
	return t.baseTab.portalColor(guid)
}

func (t *homogeneousTab) makeSelectedPortalsAnchors() {
	for guid := range t.selectedPortals {
		if _, ok := t.disabledPortals[guid]; ok {
			continue
		}
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

func (t *homogeneousTab) solutionString() string {
	return lib.HomogeneousDrawToolsString(t.depth, t.solution)
}
func (t *homogeneousTab) onPortalContextMenu(x, y int) {
	var aSelectedGuid string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	numSelectedAnchor := 0
	numSelectedNotAnchor := 0
	for guid := range t.selectedPortals {
		aSelectedGuid = guid
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
	menuHeader := fmt.Sprintf("%d portals selected", len(t.selectedPortals))
	if len(t.selectedPortals) == 1 {
		menuHeader = t.portalMap[aSelectedGuid].Name
	}
	mb := fltk.NewMenuButton(x, y, 100, 100, menuHeader)
	mb.SetCallback(func() { fmt.Println("menu Callback") })
	mb.SetType(fltk.POPUP3)
	if numSelectedDisabled > 0 {
		if len(t.selectedPortals) == 1 {
			mb.Add("Enable", func() { t.enableSelectedPortals() })
		} else {
			mb.Add("Enable All", func() { t.enableSelectedPortals() })
		}
	}
	if numSelectedEnabled > 0 {
		if len(t.selectedPortals) == 1 {
			mb.Add("Disable", func() { t.disableSelectedPortals() })
		} else {
			mb.Add("Disable All", func() { t.disableSelectedPortals() })
		}
	}
	if numSelectedAnchor > 0 {
		if len(t.selectedPortals) == 1 {
			mb.Add("Unmake anchor", func() { t.unmakeSelectedPortalsAnchors() })
		} else {
			mb.Add("Unmake all anchors", func() { t.unmakeSelectedPortalsAnchors() })
		}
	}
	if numSelectedNotAnchor > 0 && numSelectedNotAnchor+len(t.anchorPortals) <= 3 {
		if len(t.selectedPortals) == 1 {
			mb.Add("Make anchor", func() { t.makeSelectedPortalsAnchors() })
		} else {
			mb.Add("Make all anchors", func() { t.makeSelectedPortalsAnchors() })
		}
	}
	mb.Popup()
	mb.Destroy()
}
