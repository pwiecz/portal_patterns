package main

import (
	"runtime"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
	"github.com/pwiecz/portal_patterns/gui/osm"
)

func main() {
	runtime.LockOSThread()
	conf := configuration.LoadConfiguration()
	w := fltk.NewWindow(800, 600)
	w.Begin()
	t := fltk.NewTabs(10, 10, 780, 580)
	w.Resizable(t)
	tileFetcher := osm.NewMapTiles()
	newHomogeneousTab(conf, tileFetcher)
	newHerringboneTab(conf, tileFetcher)
	newDoubleHerringboneTab(conf, tileFetcher)
	newCobwebTab(conf, tileFetcher)
	newDroneFlightTab(conf, tileFetcher)
	// Mark one random tab as resizable, as per www.fltk.org/doc-1.3/classFl__Tabs.html - "resizing caveats"
	flipField := newFlipFieldTab(conf, tileFetcher)

	t.End()
	t.Resizable(flipField)
	w.End()

	fltk.Lock()
	w.Show()

	fltk.Run()
}
