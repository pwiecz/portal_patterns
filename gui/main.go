package main

import "github.com/pwiecz/atk/tk"
import "github.com/pwiecz/portal_patterns/gui/osm"
import "github.com/pwiecz/portal_patterns/configuration"

func main() {
	conf := configuration.LoadConfiguration()
	tileFetcher := osm.NewMapTiles()
	tk.MainLoop(func() {
		tk.SetMenuTearoff(false)
		mw := NewWindow(conf, tileFetcher)
		mw.SetTitle("Portal patterns")
		mw.Center()
		mw.ResizeN(640, 480)
		mw.ShowNormal()
	})
}

type Window struct {
	*tk.Window
	tab *tk.Notebook
}

func NewWindow(conf *configuration.Configuration, tileFetcher *osm.MapTiles) *Window {

	mw := &Window{}
	mw.Window = tk.RootWindow()
	mw.tab = tk.NewNotebook(mw)

	mw.tab.AddTab(NewHomogeneousTab(mw, conf, tileFetcher), "Homogeneous")
	mw.tab.AddTab(NewHerringboneTab(mw, conf, tileFetcher), "Herringbone")
	mw.tab.AddTab(NewDoubleHerringboneTab(mw, conf, tileFetcher), "Double herringbone")
	mw.tab.AddTab(NewCobwebTab(mw, conf, tileFetcher), "Cobweb")
	mw.tab.AddTab(NewDroneFlightTab(mw, conf, tileFetcher), "Drone Flight")
	mw.tab.AddTab(NewFlipFieldTab(mw, conf, tileFetcher), "Flip Field")

	vbox := tk.NewVPackLayout(mw)
	vbox.AddWidgetEx(mw.tab, tk.FillBoth, true, 0)
	return mw
}
