package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
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
	solutionText  string
	cornerPortals map[string]struct{}
}

var _ pattern = (*homogeneousTab)(nil)

func newHomogeneousTab(portals *Portals) *homogeneousTab {
	t := &homogeneousTab{}
	t.baseTab = newBaseTab("Homogeneous", portals, t)
	t.cornerPortals = make(map[string]struct{})

	maxDepthPack := fltk.NewPack(0, 0, 700, 30)
	maxDepthPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.maxDepth = fltk.NewSpinner(0, 0, 200, 30, "Max depth:")
	t.maxDepth.SetMinimum(1)
	t.maxDepth.SetMaximum(8)
	t.maxDepth.SetValue(6)
	t.maxDepth.SetType(fltk.SPINNER_INT_INPUT)
	maxDepthPack.End()
	t.Add(maxDepthPack)

	innerPortalsPack := fltk.NewPack(0, 0, 700, 30)
	innerPortalsPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.innerPortals = fltk.NewChoice(0, 0, 200, 30, "Inner portal positions:")
	t.innerPortals.Add("Arbitrary", func() {})
	t.innerPortals.Add("Spread around (slow)", func() {})
	t.innerPortals.SetValue(0)
	innerPortalsPack.End()
	t.Add(innerPortalsPack)

	topLevelPack := fltk.NewPack(0, 0, 700, 30)
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

	purePack := fltk.NewPack(0, 0, 700, 30)
	purePack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.pure = fltk.NewCheckButton(0, 0, 200, 30, "Pure")
	purePack.End()
	t.Add(purePack)

	t.End()

	return t
}

func (t *homogeneousTab) onReset() {
	t.cornerPortals = make(map[string]struct{})
	t.depth = 0
	t.solution = nil
	t.solutionText = ""
}
func (t *homogeneousTab) onSearch(progressFunc func(int, int), onSearchDone func()) {
	options := []lib.HomogeneousOption{
		lib.HomogeneousMaxDepth(t.maxDepth.Value()),
		lib.HomogeneousProgressFunc(progressFunc),
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
		corners := []int{}
		for i, portal := range portals {
			if _, ok := t.cornerPortals[portal.Guid]; ok {
				corners = append(corners, i)
			}
		}
		options = append(options, lib.HomogeneousFixedCornerIndices(corners))
		solution, depth := lib.DeepestHomogeneous(portals, options...)
		fltk.Awake(func() {
			t.solution, t.depth = solution, depth
			if t.depth > 0 {
				t.solutionText = fmt.Sprintf("Solution depth: %d", t.depth)
			} else {
				t.solutionText = "No solution found"
			}
			onSearchDone()
		})
	}()
}

func (t *homogeneousTab) hasSolution() bool {
	return len(t.solution) > 0
}
func (t *homogeneousTab) solutionInfoString() string {
	return t.solutionText
}
func (t *homogeneousTab) solutionDrawToolsString() string {
	return lib.HomogeneousDrawToolsString(t.depth, t.solution)
}
func (t *homogeneousTab) solutionPaths() [][]s2.Point {
	return portalPathsToPointPaths(lib.HomogeneousPolylines(t.depth, t.solution))
}
func (t *homogeneousTab) portalLabel(guid string) string {
	if _, ok := t.cornerPortals[guid]; ok {
		return "Corner"
	}
	return t.baseTab.portalLabel(guid)
}
func (t *homogeneousTab) portalColor(guid string) (color.Color, color.Color) {
	if _, ok := t.cornerPortals[guid]; ok {
		return color.NRGBA{0, 128, 0, 128}, t.baseTab.strokeColor(guid)
	}
	return t.baseTab.portalColor(guid)
}

func (t *homogeneousTab) enableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		delete(t.portals.disabledPortals, guid)
	}
}

func (t *homogeneousTab) disableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		t.portals.disabledPortals[guid] = struct{}{}
		delete(t.cornerPortals, guid)
	}
}

func (t *homogeneousTab) makeSelectedPortalsCorners() {
	for guid := range t.portals.selectedPortals {
		delete(t.portals.disabledPortals, guid)
		t.cornerPortals[guid] = struct{}{}
	}
}
func (t *homogeneousTab) unmakeSelectedPortalsCorners() {
	for guid := range t.portals.selectedPortals {
		delete(t.cornerPortals, guid)
	}
}

func (t *homogeneousTab) contextMenu() *menu {
	var aSelectedGUID string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	numSelectedCorner := 0
	numSelectedNotCorner := 0
	for guid := range t.portals.selectedPortals {
		aSelectedGUID = guid
		if _, ok := t.portals.disabledPortals[guid]; ok {
			numSelectedDisabled++
		} else {
			numSelectedEnabled++
		}
		if _, ok := t.cornerPortals[guid]; ok {
			numSelectedCorner++
		} else {
			numSelectedNotCorner++
		}
	}
	menu := &menu{}
	if len(t.portals.selectedPortals) > 1 {
		menu.header = fmt.Sprintf("%d portals selected", len(t.portals.selectedPortals))
	} else if len(t.portals.selectedPortals) == 1 {
		menu.header = t.portals.portalMap[aSelectedGUID].Name
	}
	if numSelectedDisabled > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Enable", t.enableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Enable All", t.enableSelectedPortals})
		}
	}
	if numSelectedEnabled > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Disable", t.disableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Disable All", t.disableSelectedPortals})
		}
	}
	if numSelectedCorner > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Unmake corner", t.unmakeSelectedPortalsCorners})
		} else {
			menu.items = append(menu.items, menuItem{"Unmake all corners", t.unmakeSelectedPortalsCorners})
		}
	}
	if numSelectedNotCorner > 0 && numSelectedNotCorner+len(t.cornerPortals) <= 3 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Make corner", t.makeSelectedPortalsCorners})
		} else {
			menu.items = append(menu.items, menuItem{"Make all corners", t.makeSelectedPortalsCorners})
		}
	}
	return menu
}

type homogeneousState struct {
	MaxDepth      int      `json:"maxDepth"`
	InnerPortals  int      `json:"innerPortals"`
	TopLevel      int      `json:"topLevel"`
	Pure          bool     `json:"pure"`
	CornerPortals []string `json:"cornerPortals"`
	Depth         int      `json:"depth"`
	Solution      []string `json:"solution"`
	SolutionText  string   `json:"solutionText"`
}

func (t *homogeneousTab) state() homogeneousState {
	state := homogeneousState{
		MaxDepth:     int(t.maxDepth.Value()),
		InnerPortals: t.innerPortals.Value(),
		TopLevel:     t.topLevel.Value(),
		Pure:         t.pure.Value(),
		Depth:        int(t.depth),
		SolutionText: t.solutionText,
	}
	for cornerGUID := range t.cornerPortals {
		state.CornerPortals = append(state.CornerPortals, cornerGUID)
	}
	for _, solutionPortal := range t.solution {
		state.Solution = append(state.Solution, solutionPortal.Guid)
	}
	return state
}

func (t *homogeneousTab) load(state homogeneousState) error {
	if state.MaxDepth <= 0 {
		return fmt.Errorf("non-positive homogeneous.maxDepth value %d", state.MaxDepth)
	}
	t.maxDepth.SetValue(float64(state.MaxDepth))
	if state.InnerPortals < 0 || state.InnerPortals >= t.innerPortals.Size() {
		return fmt.Errorf("invalid homogeneous.innerPortals value %d", state.InnerPortals)
	}
	t.innerPortals.SetValue(state.InnerPortals)
	if state.TopLevel < 0 || state.TopLevel >= t.topLevel.Size() {
		return fmt.Errorf("imvalid homogeneous.topLevel value %d", state.TopLevel)
	}
	t.topLevel.SetValue(state.TopLevel)
	t.pure.SetValue(state.Pure)
	t.cornerPortals = make(map[string]struct{})
	for _, cornerGUID := range state.CornerPortals {
		if _, ok := t.portals.portalMap[cornerGUID]; !ok {
			return fmt.Errorf("unknown homogeneous corner portal %s", cornerGUID)
		}
		t.cornerPortals[cornerGUID] = struct{}{}
	}
	if state.Depth < 0 {
		return fmt.Errorf("negative homogeneous.depth value %d", state.Depth)
	}
	t.depth = uint16(state.Depth)
	t.solution = nil
	for _, solutionGUID := range state.Solution {
		if solutionPortal, ok := t.portals.portalMap[solutionGUID]; !ok {
			return fmt.Errorf("unknown homogeneous solution portal %s", solutionGUID)
		} else {
			t.solution = append(t.solution, solutionPortal)
		}
	}
	t.solutionText = state.SolutionText
	return nil
}
