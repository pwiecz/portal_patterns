package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type homogeneousTab struct {
	*baseTab
	maxDepth     *fltk.Spinner
	innerPortals *fltk.Choice
	topLevel     *fltk.Choice
	pure         *fltk.CheckButton
	depth        uint16
	solution     []lib.Portal
}

func NewHomogeneousTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *homogeneousTab {
	t := &homogeneousTab{}
	mainPack := fltk.NewPack(20, 40, 760, 540, "Homogeneous")
	mainPack.SetType(fltk.VERTICAL)
	mainPack.SetSpacing(5)
	t.baseTab = newBaseTab("Drone Flight", configuration, tileFetcher, t)

	maxDepthPack := fltk.NewPack(0, 0, 760, 30)
	maxDepthPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.maxDepth = fltk.NewSpinner(0, 0, 200, 30, "Max depth:")
	t.maxDepth.SetMinimum(1)
	t.maxDepth.SetMaximum(8)
	t.maxDepth.SetValue(6)
	t.maxDepth.SetType(fltk.SPINNER_INT_INPUT)
	maxDepthPack.End()
	mainPack.Add(maxDepthPack)

	innerPortalsPack := fltk.NewPack(0, 0, 760, 30)
	innerPortalsPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.innerPortals = fltk.NewChoice(0, 0, 200, 30, "Inner portal positions:")
	t.innerPortals.Add("Arbitrary", func() {})
	t.innerPortals.Add("Spread around (slow)", func() {})
	t.innerPortals.SetValue(0)
	innerPortalsPack.End()
	mainPack.Add(innerPortalsPack)

	topLevelPack := fltk.NewPack(0, 0, 760, 30)
	topLevelPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.topLevel = fltk.NewChoice(0, 0, 200, 30, "Top level triangle:")
	t.topLevel.Add("Arbitrary", func() {})
	t.topLevel.Add("Smallest area", func() {})
	t.topLevel.Add("Largest area", func() {})
	t.topLevel.Add("Most Equilateral", func() {})
	t.topLevel.Add("Random", func() {})
	t.topLevel.SetValue(0)
	topLevelPack.End()
	mainPack.Add(topLevelPack)

	purePack := fltk.NewPack(0, 0, 760, 30)
	purePack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.pure = fltk.NewCheckButton(0, 0, 200, 30, "Pure")
	purePack.End()
	mainPack.Add(purePack)

	mainPack.Add(t.searchSaveCopyPack)
	mainPack.Add(t.progress)
	if t.portalList != nil {
		mainPack.Add(t.portalList)
		mainPack.Resizable(t.portalList)
	}
	mainPack.End()

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
	case 1:
		options = append(options, lib.HomogeneousSmallestArea{})
	case 2:
		options = append(options, lib.HomogeneousLargestArea{})
	case 3:
		options = append(options, lib.HomogeneousMostEquilateralTriangle{})
	case 4:
		rand := rand.New(rand.NewSource(time.Now().UnixNano()))
		options = append(options, lib.HomogeneousRandom{Rand: rand})
	}
	go func() {
		t.solution, t.depth = lib.DeepestHomogeneous(t.portals, options...)
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
	/*	if t.disabledPortals[guid] {
			return "Disabled"
		}
		if t.anchorPortals[guid] {
			return "Anchor"
		}*/
	return "Normal"
}

func (t *homogeneousTab) portalColor(guid string) string {
	return ""
}
func (t *homogeneousTab) solutionString() string {
	return lib.HomogeneousDrawToolsString(t.depth, t.solution)
}
func (t *homogeneousTab) onPortalContextMenu(guid string, x, y int) {}
