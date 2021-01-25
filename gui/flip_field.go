package main

import (
	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/lib"
)

type FlipFieldTab struct {
	configuration      *configuration.Configuration
	numBackbonePortals *fltk.Spinner
	exactly            *fltk.CheckButton
	maxFlipPortals     *fltk.Spinner
	simpleBackbone     *fltk.CheckButton
	search             *fltk.Button
	addPortals         *fltk.Button
	progress           *fltk.Progress
	portalList         *fltk.TableRow
	mapWindow          *MapWindow
	portals            []lib.Portal
}

func NewFlipFieldTab(configuration *configuration.Configuration) *FlipFieldTab {
	t := &FlipFieldTab{
		configuration: configuration,
	}
	flipField := fltk.NewGroup(20, 30, 760, 550, "Flip Field")
	y := 40
	t.numBackbonePortals = fltk.NewSpinner(200, y, 200, 30, "Num backbone portals:")
	t.numBackbonePortals.SetType(fltk.SPINNER_INT_INPUT)
	t.numBackbonePortals.SetValue(16)
	t.exactly = fltk.NewCheckButton(440, y, 200, 30, "Exactly")
	y += 35
	t.maxFlipPortals = fltk.NewSpinner(200, y, 200, 30, "Max flip portals:")
	t.maxFlipPortals.SetType(fltk.SPINNER_INT_INPUT)
	t.maxFlipPortals.SetValue(9999)
	y += 35
	t.simpleBackbone = fltk.NewCheckButton(200, y, 200, 30, "Simple backbone")
	t.simpleBackbone.SetValue(false)
	y += 35
	buttonPack := fltk.NewPack(20, y, 200, 30)
	buttonPack.SetType(fltk.HORIZONTAL)
	buttonPack.SetSpacing(5)
	t.search = fltk.NewButton(0, 0, 80, 30, "Search")
	t.search.SetCallback(func() { t.OnSearchPressed() })
	t.search.Deactivate()
	t.addPortals = fltk.NewButton(0, 0, 100, 30, "Add portals")
	buttonPack.End()
	y += 35
	t.addPortals.SetCallback(func() { t.OnAddPortalsPressed() })
	t.progress = fltk.NewProgress(20, y, 740, 30, "")
	t.progress.SetSelectionColor(0x0000ffff)
	y += 35
	t.portalList = fltk.NewTableRow(20, y, 740, 550-10-y, func(context fltk.TableContext, r, c, x, y, w, h int) {
		t.PortalListDrawCallback(context, r, c, x, y, w, h)
	})
	t.portalList.EnableColumnHeaders()
	t.portalList.AllowColumnResizing()
	t.portalList.SetColumnWidth(0, 200)

	flipField.End()

	flipField.Resizable(t.portalList)
	return t
}

func (t *FlipFieldTab) OnSearchPressed() {
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
		backbone, rest := lib.LargestFlipField(t.portals, options...)
		if t.mapWindow != nil {
			lines := [][]lib.Portal{backbone}
			if len(rest) > 0 {
				hull := s2.NewConvexHullQuery()
				for _, p := range rest {
					hull.AddPoint(s2.PointFromLatLng(p.LatLng))
				}
				hullPoints := hull.ConvexHull().Vertices()
				if len(hullPoints) > 0 {
					hullPoints = append(hullPoints, hullPoints[0])
				}
				//				lines = append(lines, hullPoints)
			}
			t.mapWindow.SetPaths(lines)
		}
	}()
}

func (t *FlipFieldTab) OnAddPortalsPressed() {
	filename, ok := fltk.ChooseFile(
		"Select portals file",
		"JSON files (*.json)\tCSV files (*.csv)", t.configuration.PortalsDirectory, false)
	if !ok {
		return
	}
	if t.mapWindow == nil {
		t.mapWindow = NewMapWindow("Flip Field")
	} else {
		t.mapWindow.Show()
	}
	portals, _ := lib.ParseFile(filename)
	t.portals = portals
	t.portalList.SetRowCount(len(t.portals))
	t.portalList.SetColumnCount(2)
	t.mapWindow.SetPortals(t.portals)
	if len(t.portals) > 0 {
		t.search.Activate()
	} else {
		t.search.Deactivate()
	}
}

func (t *FlipFieldTab) PortalListDrawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
	switch context {
	case fltk.ContextCell:
		if row >= len(t.portals) {
			return
		}
		fltk.DrawBox(fltk.THIN_UP_BOX, x, y, w, h, 0xffffffff)
		fltk.Color(0x00000000)
		if column == 0 {
			fltk.Draw(t.portals[row].Name, x, y, w, h, fltk.ALIGN_LEFT)
		} else if column == 1 {
			fltk.Draw("-", x, y, w, h, fltk.ALIGN_CENTER)
		}
	case fltk.ContextColHeader:
		fltk.DrawBox(fltk.UP_BOX, x, y, w, h, 0x8f8f8fff)
		fltk.Color(0x00000000)
		if column == 0 {
			fltk.Draw("Name", x, y, w, h, fltk.ALIGN_CENTER)
		} else if column == 1 {
			fltk.Draw("State", x, y, w, h, fltk.ALIGN_CENTER)
		}
	}
}
