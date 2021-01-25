package main

import (
	"runtime"

	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"
)

func main() {
	runtime.LockOSThread()
	conf := configuration.LoadConfiguration()
	w := fltk.NewWindow(800, 600)
	w.Begin()
	t := fltk.NewTabs(10, 10, 780, 580)
	w.Resizable(t)
	NewHomogeneousTab(conf)
	NewHerringboneTab(conf)
	NewDoubleHerringboneTab(conf)
	NewCobwebTab(conf)
	NewDroneFlightTab(conf)
	NewFlipFieldTab(conf)

	t.End()
	w.End()

	fltk.Lock()
	w.Show()

	fltk.Run()
}
