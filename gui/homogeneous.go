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
	t.topLevel.Add("Arbitrary", func() {})
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
		portals := t.enabledPortals()
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

func (t *homogeneousTab) solutionString() string {
	return lib.HomogeneousDrawToolsString(t.depth, t.solution)
}
func (t *homogeneousTab) onPortalContextMenu(x, y int) {
	var aSelectedGuid string
	allSelectedEnabled := true
	allSelectedDisabled := true
	for guid := range t.selectedPortals {
		aSelectedGuid = guid
		if _, ok := t.disabledPortals[guid]; ok {
			allSelectedEnabled = false
		} else {
			allSelectedDisabled = false
		}
	}
	menuHeader := fmt.Sprintf("%d portals selected", len(t.selectedPortals))
	if len(t.selectedPortals) == 1 {
		menuHeader = t.portalMap[aSelectedGuid].Name
	}
	mb := fltk.NewMenuButton(x, y, 100, 100, menuHeader)
	mb.SetCallback(func() { fmt.Println("menu Callback") })
	mb.SetType(fltk.POPUP3)
	if !allSelectedEnabled {
		if len(t.selectedPortals) == 1 {
			mb.Add("Enable", func() { t.enableSelectedPortals() })
		} else {
			mb.Add("Enable All", func() { t.enableSelectedPortals() })
		}
	}
	if !allSelectedDisabled {
		if len(t.selectedPortals) == 1 {
			mb.Add("Disable", func() { t.disableSelectedPortals() })
		} else {
			mb.Add("Disable All", func() { t.disableSelectedPortals() })
		}
	}
	mb.Popup()
	mb.Destroy()
}
