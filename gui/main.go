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
	tabs := container.NewAppTabs(
		NewHomogeneousTab(a, w, conf, tileFetcher),
		NewHerringboneTab(a, w, conf, tileFetcher),
		NewDoubleHerringboneTab(a, w, conf, tileFetcher),
		NewCobwebTab(a, w, conf, tileFetcher),
		NewDroneFlightTab(a, w, conf, tileFetcher),
		NewFlipFieldTab(a, w, conf, tileFetcher))
	w.SetContent(tabs)
	w.ShowAndRun()
}
