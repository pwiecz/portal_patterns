package main

import (
	"fmt"

	//	"github.com/golang/geo/s2"
	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type flipFieldTab struct {
	*baseTab
	numBackbonePortals *fltk.Spinner
	exactly            *fltk.CheckButton
	maxFlipPortals     *fltk.Spinner
	simpleBackbone     *fltk.CheckButton
	backbone           []lib.Portal
	rest               []lib.Portal
}

var _ = (*flipFieldTab)(nil)

func NewFlipFieldTab(configuration *configuration.Configuration, tileFetcher *osm.MapTiles) *flipFieldTab {
	t := &flipFieldTab{}
	t.baseTab = newBaseTab("Flip Field", configuration, tileFetcher, t)

	numBackbonePortalsPack := fltk.NewPack(0, 0, 760, 30)
	numBackbonePortalsPack.SetType(fltk.HORIZONTAL)
	numBackbonePortalsPack.SetSpacing(5)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 195, 30)
	t.numBackbonePortals = fltk.NewSpinner(0, 0, 200, 30, "Num backbone portals:")
	t.numBackbonePortals.SetType(fltk.SPINNER_INT_INPUT)
	t.numBackbonePortals.SetValue(16)
	t.exactly = fltk.NewCheckButton(0, 0, 200, 30, "Exactly")
	numBackbonePortalsPack.End()
	t.Add(numBackbonePortalsPack)

	maxFlipPortalsPack := fltk.NewPack(0, 0, 760, 30)
	maxFlipPortalsPack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.maxFlipPortals = fltk.NewSpinner(0, 0, 200, 30, "Max flip portals:")
	t.maxFlipPortals.SetType(fltk.SPINNER_INT_INPUT)
	t.maxFlipPortals.SetValue(9999)
	maxFlipPortalsPack.End()
	t.Add(maxFlipPortalsPack)

	simpleBackbonePack := fltk.NewPack(0, 0, 760, 30)
	simpleBackbonePack.SetType(fltk.HORIZONTAL)
	fltk.NewBox(fltk.NO_BOX, 0, 0, 200, 30)
	t.simpleBackbone = fltk.NewCheckButton(0, 0, 200, 30, "Simple backbone")
	t.simpleBackbone.SetValue(false)
	simpleBackbonePack.End()
	t.Add(simpleBackbonePack)

	t.Add(t.searchSaveCopyPack)
	t.Add(t.progress)
	if t.portalList != nil {
		t.Add(t.portalList)
	}
	t.End()

	return t
}

func (t *flipFieldTab) onReset() {
	t.backbone = nil
	t.rest = nil
}
func (t *flipFieldTab) onSearch() {
	progressFunc := func(val, max int) {
		fltk.Awake(func() {
			t.progress.SetMaximum(float64(max))
			t.progress.SetValue(float64(val))
		})
	}
	numPortalLimit := lib.LESS_EQUAL
	if t.exactly.Value() {
		numPortalLimit = lib.EQUAL
	}
	options := []lib.FlipFieldOption{
		lib.FlipFieldProgressFunc(progressFunc),
		lib.FlipFieldBackbonePortalLimit{Value: int(t.numBackbonePortals.Value()), LimitType: numPortalLimit},
		lib.FlipFieldMaxFlipPortals(int(t.maxFlipPortals.Value())),
		lib.FlipFieldSimpleBackbone(t.simpleBackbone.Value()),
	}
	go func() {
		portals := t.enabledPortals()
		t.backbone, t.rest = lib.LargestFlipField(portals, options...)
		if t.mapWindow != nil {
			lines := [][]s2.Point{portalsToPoints(t.backbone)}
			if len(t.rest) > 0 {
				hull := s2.NewConvexHullQuery()
				for _, p := range t.rest {
					hull.AddPoint(s2.PointFromLatLng(p.LatLng))
				}
				hullPoints := hull.ConvexHull().Vertices()
				if len(hullPoints) > 0 {
					hullPoints = append(hullPoints, hullPoints[0])
				}
				lines = append(lines, hullPoints)
			}
			t.mapWindow.SetPaths(lines)
		}
		fltk.Awake(func() {
			solutionText := fmt.Sprintf("Num backbone portals: %d, num flip portals: %d", len(t.backbone), len(t.rest))
			t.onSearchDone(solutionText)
		})
	}()
}

func (t *flipFieldTab) solutionString() string {
	s := fmt.Sprintf("[%s", lib.PolylineFromPortalList(t.backbone))
	if len(t.rest) > 0 {
		s += fmt.Sprintf(",%s", lib.MarkersFromPortalList(t.rest))
	}
	return s + "]"
}

func (t *flipFieldTab) enableSelectedPortals() {
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

func (t *flipFieldTab) disableSelectedPortals() {
	for guid := range t.selectedPortals {
		t.disabledPortals[guid] = struct{}{}
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

func (t *flipFieldTab) contextMenu() *menu {
	var aSelectedGuid string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	for guid := range t.selectedPortals {
		aSelectedGuid = guid
		if _, ok := t.disabledPortals[guid]; ok {
			numSelectedDisabled++
		} else {
			numSelectedEnabled++
		}
	}
	menu := &menu{}
	if len(t.selectedPortals) > 1 {
		menu.header = fmt.Sprintf("%d portals selected", len(t.selectedPortals))
	} else if len(t.selectedPortals) == 1 {
		menu.header = t.portalMap[aSelectedGuid].Name
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
	return menu
}
