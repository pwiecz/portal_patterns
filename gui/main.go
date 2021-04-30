package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
)

func main() {
	a := app.New()
	w := a.NewWindow("Portal Patterns")
	conf := configuration.LoadConfiguration()
	tileFetcher := osm.NewMapTiles()
	tabs := container.NewAppTabs(NewHomogeneousTab(a, w, conf, tileFetcher))
	w.SetContent(tabs)
	w.ShowAndRun()
}

// type Window struct {
// 	*tk.Window
// 	tab *tk.Notebook
// }

// func NewWindow(conf *configuration.Configuration, tileFetcher *osm.MapTiles) *Window {

// 	mw := &Window{}
// 	mw.Window = tk.RootWindow()
// 	mw.tab = tk.NewNotebook(mw)

// 	//	mw.tab.AddTab(NewHomogeneousTab(mw, conf, tileFetcher), "Homogeneous")
// 	mw.tab.AddTab(NewHerringboneTab(mw, conf, tileFetcher), "Herringbone")
// 	mw.tab.AddTab(NewDoubleHerringboneTab(mw, conf, tileFetcher), "Double herringbone")
// 	mw.tab.AddTab(NewCobwebTab(mw, conf, tileFetcher), "Cobweb")
// 	mw.tab.AddTab(NewDroneFlightTab(mw, conf, tileFetcher), "Drone Flight")
// 	mw.tab.AddTab(NewFlipFieldTab(mw, conf, tileFetcher), "Flip Field")

// 	vbox := tk.NewVPackLayout(mw)
// 	vbox.AddWidgetEx(mw.tab, tk.FillBoth, true, 0)
// 	return mw
// }
