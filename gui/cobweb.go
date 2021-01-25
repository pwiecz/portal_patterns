package main

import (
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/lib"
)

type CobwebTab struct {
	configuration *configuration.Configuration
	search        *fltk.Button
	addPortals    *fltk.Button
	progress      *fltk.Progress
	portalList    *fltk.TableRow
	mapWindow     *MapWindow
	portals       []lib.Portal
}

func NewCobwebTab(configuration *configuration.Configuration) *CobwebTab {
	t := &CobwebTab{
		configuration: configuration,
	}
	cobweb := fltk.NewGroup(20, 30, 760, 550, "Cobweb")
	y := 40
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

	cobweb.End()

	cobweb.Resizable(t.portalList)
	return t
}

func (t *CobwebTab) OnSearchPressed() {
	progressFunc := func(val, max int) {
		fltk.Awake(func() {
			t.progress.SetMaximum(float64(max))
			t.progress.SetValue(float64(val))
		})
	}
	go func() {
		result := lib.LargestCobweb(t.portals, []int{}, progressFunc)
		if t.mapWindow != nil {
			t.mapWindow.SetPaths([][]lib.Portal{lib.CobwebPolyline(result)})
		}
	}()
}

func (t *CobwebTab) OnAddPortalsPressed() {
	filename, ok := fltk.ChooseFile(
		"Select portals file",
		"JSON files (*.json)\tCSV files (*.csv)", t.configuration.PortalsDirectory, false)
	if !ok {
		return
	}
	if t.mapWindow == nil {
		t.mapWindow = NewMapWindow("Cobweb")
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

func (t *CobwebTab) PortalListDrawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
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
