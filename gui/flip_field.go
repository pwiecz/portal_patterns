package main

import (
	"fmt"
	"runtime"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/lib"
)

type flipFieldTab struct {
	*baseTab
	numBackbonePortals *fltk.Spinner
	exactly            *fltk.CheckButton
	maxFlipPortals     *fltk.Spinner
	simpleBackbone     *fltk.CheckButton
	backbone           []lib.Portal
	flipPortals        []lib.Portal
	solutionText       string
}

var _ pattern = (*flipFieldTab)(nil)

func newFlipFieldTab(portals *Portals) *flipFieldTab {
	t := &flipFieldTab{}
	t.baseTab = newBaseTab("Flip Field", portals, t)

	warningLabel := fltk.NewBox(fltk.NO_BOX, 0, 0, 700, 30)
	warningLabel.SetAlign(fltk.ALIGN_INSIDE | fltk.ALIGN_LEFT)
	warningLabel.SetLabel("* WARNING: a greedy algorithm which finds suboptimal solutions *")
	t.Add(warningLabel)

	numBackbonePortalsPack := fltk.NewPack(0, 0, 700, 30)
	numBackbonePortalsPack.SetType(fltk.HORIZONTAL)
	numBackbonePortalsPack.SetSpacing(5)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 195, 30)
	t.numBackbonePortals = fltk.NewSpinner(0, 0, 200, 30, "Num backbone portals:")
	t.numBackbonePortals.SetType(fltk.SPINNER_INT_INPUT)
	t.numBackbonePortals.SetValue(16)
	t.exactly = fltk.NewCheckButton(0, 0, 200, 30, "Exactly")
	numBackbonePortalsPack.End()
	t.Add(numBackbonePortalsPack)

	maxFlipPortalsPack := fltk.NewPack(0, 0, 700, 30)
	maxFlipPortalsPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.maxFlipPortals = fltk.NewSpinner(0, 0, 200, 30, "Max flip portals:")
	t.maxFlipPortals.SetType(fltk.SPINNER_INT_INPUT)
	t.maxFlipPortals.SetValue(9999)
	maxFlipPortalsPack.End()
	t.Add(maxFlipPortalsPack)

	simpleBackbonePack := fltk.NewPack(0, 0, 700, 30)
	simpleBackbonePack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.simpleBackbone = fltk.NewCheckButton(0, 0, 200, 30, "Simple backbone")
	t.simpleBackbone.SetValue(false)
	simpleBackbonePack.End()

	t.Add(simpleBackbonePack)

	t.End()

	return t
}

func (t *flipFieldTab) onReset() {
	t.backbone = nil
	t.flipPortals = nil
	t.solutionText = ""
}
func (t *flipFieldTab) onSearch(progressFunc func(int, int), onSearchDone func()) {
	numPortalLimit := lib.LESS_EQUAL
	if t.exactly.Value() {
		numPortalLimit = lib.EQUAL
	}
	options := []lib.FlipFieldOption{
		lib.FlipFieldProgressFunc(progressFunc),
		lib.FlipFieldBackbonePortalLimit{Value: int(t.numBackbonePortals.Value()), LimitType: numPortalLimit},
		lib.FlipFieldMaxFlipPortals(int(t.maxFlipPortals.Value())),
		lib.FlipFieldSimpleBackbone(t.simpleBackbone.Value()),
		lib.FlipFieldNumWorkers(runtime.GOMAXPROCS(0)),
	}
	portals := t.enabledPortals()
	go func() {
		backbone, flipPortals := lib.LargestFlipField(portals, options...)
		fltk.Awake(func() {
			t.backbone, t.flipPortals = backbone, flipPortals
			t.solutionText = fmt.Sprintf("Num backbone portals: %d, num flip portals: %d", len(t.backbone), len(t.flipPortals))
			onSearchDone()
		})
	}()
}

func (t *flipFieldTab) hasSolution() bool {
	return len(t.backbone) > 0 && len(t.flipPortals) > 0
}
func (t *flipFieldTab) solutionInfoString() string {
	return t.solutionText
}
func (t *flipFieldTab) solutionDrawToolsString() string {
	s := fmt.Sprintf("[%s", lib.PolylineFromPortalList(t.backbone))
	if len(t.flipPortals) > 0 {
		s += fmt.Sprintf(",%s", lib.MarkersFromPortalList(t.flipPortals))
	}
	return s + "]"
}
func (t *flipFieldTab) solutionPaths() [][]s2.Point {
	lines := [][]s2.Point{portalsToPoints(t.backbone)}
	if len(t.flipPortals) > 0 {
		hull := s2.NewConvexHullQuery()
		for _, p := range t.flipPortals {
			hull.AddPoint(s2.PointFromLatLng(p.LatLng))
		}
		hullPoints := hull.ConvexHull().Vertices()
		if len(hullPoints) > 0 {
			hullPoints = append(hullPoints, hullPoints[0])
		}
		lines = append(lines, hullPoints)
	}
	return lines
}

func (t *flipFieldTab) enableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		delete(t.portals.disabledPortals, guid)
	}
}

func (t *flipFieldTab) disableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		t.portals.disabledPortals[guid] = struct{}{}
	}
}

func (t *flipFieldTab) contextMenu() *menu {
	var aSelectedGUID string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	for guid := range t.portals.selectedPortals {
		aSelectedGUID = guid
		if _, ok := t.portals.disabledPortals[guid]; ok {
			numSelectedDisabled++
		} else {
			numSelectedEnabled++
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
	return menu
}

type flipFieldState struct {
	NumBackbonePortals int      `json:"numBackbonePortals"`
	Exactly            bool     `json:"exactly"`
	MaxFlipPortals     int      `json:"maxFlipPortals"`
	SimpleBackbone     bool     `json:"simpleBackbone"`
	Backbone           []string `json:"backbone"`
	FlipPortals        []string `json:"flipPortals"`
	SolutionText       string   `json:"solutionText"`
}

func (t *flipFieldTab) state() flipFieldState {
	state := flipFieldState{
		NumBackbonePortals: int(t.numBackbonePortals.Value()),
		Exactly:            t.exactly.Value(),
		MaxFlipPortals:     int(t.maxFlipPortals.Value()),
		SimpleBackbone:     t.simpleBackbone.Value(),
		SolutionText:       t.solutionText,
	}
	for _, backbonePortal := range t.backbone {
		state.Backbone = append(state.Backbone, backbonePortal.Guid)
	}
	for _, flipPortal := range t.flipPortals {
		state.FlipPortals = append(state.FlipPortals, flipPortal.Guid)
	}
	return state
}

func (t *flipFieldTab) load(state flipFieldState) error {
	if state.NumBackbonePortals <= 0 {
		return fmt.Errorf("non-positive flipField.numBackbonePortals value %d", state.NumBackbonePortals)
	}
	t.numBackbonePortals.SetValue(float64(state.NumBackbonePortals))
	t.exactly.SetValue(state.Exactly)
	if state.MaxFlipPortals <= 0 {
		return fmt.Errorf("non-positive flipField.maxFlipPortals value %d", state.MaxFlipPortals)
	}
	t.maxFlipPortals.SetValue(float64(state.MaxFlipPortals))
	t.simpleBackbone.SetValue(state.SimpleBackbone)
	t.backbone = nil
	for _, backboneGUID := range state.Backbone {
		if backbonePortal, ok := t.portals.portalMap[backboneGUID]; !ok {
			return fmt.Errorf("invalid flipField backbone portal \"%s\"", backboneGUID)
		} else {
			t.backbone = append(t.backbone, backbonePortal)
		}
	}
	for _, flipPortalGUID := range state.FlipPortals {
		if flipPortal, ok := t.portals.portalMap[flipPortalGUID]; !ok {
			return fmt.Errorf("invalid flipField flip portal \"%s\"", flipPortalGUID)
		} else {
			t.flipPortals = append(t.flipPortals, flipPortal)
		}
	}
	t.solutionText = state.SolutionText
	return nil
}
