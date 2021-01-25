package main

import (
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/lib"
)

type DroneFlightTab struct {
	configuration *configuration.Configuration
	useLongJumps  *fltk.CheckButton
	optimizeFor   *fltk.Choice
	search        *fltk.Button
	addPortals    *fltk.Button
	progress      *fltk.Progress
	portalList    *fltk.TableRow
	mapWindow     *MapWindow
	portals       []lib.Portal
}

func NewDroneFlightTab(configuration *configuration.Configuration) *DroneFlightTab {
	t := &DroneFlightTab{
		configuration: configuration,
	}
	droneFlight := fltk.NewGroup(20, 30, 760, 550, "Drone Flight")
	y := 40
	t.useLongJumps = fltk.NewCheckButton(200, y, 200, 30, "Use long jumps (key needed)")
	t.useLongJumps.SetValue(true)
	y += 35
	t.optimizeFor = fltk.NewChoice(200, y, 200, 30, "Optimize for:")
	t.optimizeFor.Add("Least keys needed", func() {})
	t.optimizeFor.Add("Least jump", func() {})
	t.optimizeFor.SetValue(0)
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

	droneFlight.End()

	droneFlight.Resizable(t.portalList)
	return t
}

func (t *DroneFlightTab) OnSearchPressed() {
	progressFunc := func(val, max int) {
		fltk.Awake(func() {
			t.progress.SetMaximum(float64(max))
			t.progress.SetValue(float64(val))
		})
	}
	options := []lib.DroneFlightOption{
		lib.DroneFlightProgressFunc(progressFunc),
		lib.DroneFlightUseLongJumps(t.useLongJumps.Value()),
	}
	switch t.optimizeFor.Value() {
	case 0:
		options = append(options, lib.DroneFlightLeastKeys{})
	case 1:
		options = append(options, lib.DroneFlightLeastJumps{})
	}
	go func() {
		result, _ := lib.LongestDroneFlight(t.portals, options...)
		if t.mapWindow != nil {
			t.mapWindow.SetPaths([][]lib.Portal{lib.CobwebPolyline(result)})
		}
	}()
}

func (t *DroneFlightTab) OnAddPortalsPressed() {
	filename, ok := fltk.ChooseFile(
		"Select portals file",
		"JSON files (*.json)\tCSV files (*.csv)", t.configuration.PortalsDirectory, false)
	if !ok {
		return
	}
	if t.mapWindow == nil {
		t.mapWindow = NewMapWindow("Drone Flight")
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

func (t *DroneFlightTab) PortalListDrawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
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
