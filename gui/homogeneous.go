package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	guigl "github.com/pwiecz/portal_patterns/gui/gl"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type HomogeneousTab struct {
	configuration *configuration.Configuration
	maxDepth      *fltk.Spinner
	innerPortals  *fltk.Choice
	topLevel      *fltk.Choice
	pure          *fltk.CheckButton
	search        *fltk.Button
	addPortals    *fltk.Button
	progress      *fltk.Progress
	portalList    *fltk.TableRow
	mapDrawer     *guigl.MapWindow
	portals       []lib.Portal
}

func NewHomogeneousTab(configuration *configuration.Configuration) *HomogeneousTab {
	t := &HomogeneousTab{
		configuration: configuration,
	}
	homogeneous := fltk.NewGroup(20, 30, 760, 550, "Homogeneous")
	y := 40
	t.maxDepth = fltk.NewSpinner(200, y, 200, 30, "Max depth:")
	t.maxDepth.SetMinimum(1)
	t.maxDepth.SetMaximum(8)
	t.maxDepth.SetValue(6)
	t.maxDepth.SetType(fltk.SPINNER_INT_INPUT)
	y += 35
	t.innerPortals = fltk.NewChoice(200, y, 200, 30, "Inner portal positions:")
	t.innerPortals.Add("Arbitrary", func() {})
	t.innerPortals.Add("Spread around (slow)", func() {})
	t.innerPortals.SetValue(0)
	y += 35
	t.topLevel = fltk.NewChoice(200, y, 200, 30, "Top level triangle:")
	t.topLevel.Add("Arbitrary", func() {})
	t.topLevel.Add("Smallest area", func() {})
	t.topLevel.Add("Largest area", func() {})
	t.topLevel.Add("Most Equilateral", func() {})
	t.topLevel.Add("Random", func() {})
	t.topLevel.SetValue(0)
	y += 35
	t.pure = fltk.NewCheckButton(200, y, 200, 30, "Pure")
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

	homogeneous.End()

	homogeneous.Resizable(t.portalList)
	return t
}

func (t *HomogeneousTab) OnSearchPressed() {
	fmt.Println("Homogeneous: ", t.maxDepth.Value(), t.topLevel.Value(), t.pure.Value())
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
		solution, depth := lib.DeepestHomogeneous(t.portals, options...)
		if t.mapDrawer != nil {
			t.mapDrawer.SetPaths(lib.HomogeneousPolylines(depth, solution))
		}
	}()
}

func (t *HomogeneousTab) OnAddPortalsPressed() {
	filename, ok := fltk.ChooseFile(
		"Select portals file",
		"JSON files (*.json)\tCSV files (*.csv)", t.configuration.PortalsDirectory, false)
	if !ok {
		fmt.Println("Cancelled")
		return
	}
	if t.mapDrawer == nil {
		mw := fltk.NewWindow(800, 600)
		mw.Begin()
		tileFetcher := osm.NewMapTiles()
		t.mapDrawer = guigl.NewMapWindow("map", tileFetcher)
		var glWindow *fltk.GlWindow
		glWindow = fltk.NewGlWindow(0, 0, 800, 600, func() {
			DrawMap(glWindow, t.mapDrawer)
		})
		var prevX, prevY int
		glWindow.SetEventHandler(func(event fltk.Event) bool {
			switch event {
			case fltk.PUSH:
				prevX, prevY = fltk.EventX(), fltk.EventY()
				// return true to receive drag events
				return true
			case fltk.FOCUS:
				// return true to receive keyboard events
				return true
			case fltk.DRAG:
				if fltk.EventButton1() {
					currX, currY := fltk.EventX(), fltk.EventY()
					t.mapDrawer.Drag(prevX-currX, prevY-currY)
					prevX, prevY = currX, currY
					fltk.Awake(func() { glWindow.Redraw() })
					return true
				}
			case fltk.MOUSEWHEEL:
				dy := fltk.EventDY()
				if dy < 0 {
					t.mapDrawer.ZoomIn(fltk.EventX(), fltk.EventY())
					fltk.Awake(func() { glWindow.Redraw() })
					return true
				} else if dy > 0 {
					t.mapDrawer.ZoomOut(fltk.EventX(), fltk.EventY())
					fltk.Awake(func() { glWindow.Redraw() })
					return true
				}
			case fltk.KEY:
				if (fltk.EventState()&fltk.CTRL) != 0 &&
					(fltk.EventKey() == '+' || fltk.EventKey() == '=') {
					t.mapDrawer.ZoomIn(glWindow.W()/2, glWindow.H()/2)
					fltk.Awake(func() { glWindow.Redraw() })
					return true
				} else if fltk.EventKey() == '-' && (fltk.EventState()&fltk.CTRL) != 0 {
					t.mapDrawer.ZoomOut(glWindow.W()/2, glWindow.H()/2)
					fltk.Awake(func() { glWindow.Redraw() })
					return true
				}
			}
			return false
		})
		mw.End()
		t.mapDrawer.OnMapChanged(
			func() {
				fmt.Println("map changed")
				fltk.Awake(func() { glWindow.Redraw() })
			})
		mw.Show()
	}
	portals, _ := lib.ParseFile(filename)
	t.portals = portals
	t.portalList.SetRowCount(len(t.portals))
	t.portalList.SetColumnCount(2)
	t.mapDrawer.SetPortals(t.portals)
	if len(t.portals) > 0 {
		t.search.Activate()
	} else {
		t.search.Deactivate()
	}
}

func (t *HomogeneousTab) PortalListDrawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
	switch context {
	case fltk.ContextCell:
		if row >= len(t.portals) {
			return
		}
		//fmt.Println("drawing portal", t.portals[row].Name, "column:", column)
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
