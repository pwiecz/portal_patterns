package main

import (
	"image/color"
	"log"

	gogl "github.com/go-gl/gl/v2.1/gl"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/gui/gl"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type MapWindow struct {
	window       *fltk.Window
	glWindow     *fltk.GlWindow
	mapDrawer    *gl.MapDrawer
	prevX, prevY int
}

func NewMapWindow(title string, tileFetcher *osm.MapTiles) *MapWindow {
	w := &MapWindow{}
	w.window = fltk.NewWindow(800, 600)
	w.window.SetLabel(title + " - © OpenStreetMap")
	w.window.Begin()
	w.mapDrawer = gl.NewMapDrawer(tileFetcher)
	w.glWindow = fltk.NewGlWindow(0, 0, 800, 600, func() { w.drawMap() })
	w.glWindow.SetEventHandler(func(event fltk.Event) bool { return w.handleEvent(event) })
	w.window.End()
	w.mapDrawer.OnMapChanged(
		func() {
			fltk.Awake(func() { w.glWindow.Redraw() })
		})
	w.window.Show()
	return w
}

func (w *MapWindow) Hide() {
	w.window.Hide()
}
func (w *MapWindow) Show() {
	w.window.Show()
}
func (w *MapWindow) SetPortals(portals []lib.Portal) {
	w.mapDrawer.SetPortals(portals)
}
func (w *MapWindow) SetPaths(paths [][]lib.Portal) {
	w.mapDrawer.SetPaths(paths)
}
func (w *MapWindow) SetPortalColor(guid string, color color.Color) {
	w.mapDrawer.SetPortalColor(guid, color)
}
func (w *MapWindow) drawMap() {
	if !w.glWindow.Valid() {
		if err := gogl.Init(); err != nil {
			log.Fatal("Cannot initialize OpenGL", err)
		}
	}
	if !w.glWindow.ContextValid() {
		w.mapDrawer.Init()
	}

	w.mapDrawer.Update()
}

func (w *MapWindow) handleEvent(event fltk.Event) bool {
	switch event {
	case fltk.PUSH:
		w.prevX, w.prevY = fltk.EventX(), fltk.EventY()
		// return true to receive drag events
		return true
	case fltk.FOCUS:
		// return true to receive keyboard events
		return true
	case fltk.DRAG:
		if fltk.EventButton1() {
			currX, currY := fltk.EventX(), fltk.EventY()
			w.mapDrawer.Drag(w.prevX-currX, w.prevY-currY)
			w.prevX, w.prevY = currX, currY
			fltk.Awake(func() { w.glWindow.Redraw() })
			return true
		}
	case fltk.MOUSEWHEEL:
		dy := fltk.EventDY()
		if dy < 0 {
			w.mapDrawer.ZoomIn(fltk.EventX(), fltk.EventY())
			fltk.Awake(func() { w.glWindow.Redraw() })
			return true
		} else if dy > 0 {
			w.mapDrawer.ZoomOut(fltk.EventX(), fltk.EventY())
			fltk.Awake(func() { w.glWindow.Redraw() })
			return true
		}
	case fltk.KEY:
		if (fltk.EventState()&fltk.CTRL) != 0 &&
			(fltk.EventKey() == '+' || fltk.EventKey() == '=') {
			w.mapDrawer.ZoomIn(w.glWindow.W()/2, w.glWindow.H()/2)
			fltk.Awake(func() { w.glWindow.Redraw() })
			return true
		} else if fltk.EventKey() == '-' && (fltk.EventState()&fltk.CTRL) != 0 {
			w.mapDrawer.ZoomOut(w.glWindow.W()/2, w.glWindow.H()/2)
			fltk.Awake(func() { w.glWindow.Redraw() })
			return true
		}
	case fltk.MOVE:
		w.mapDrawer.Hover(fltk.EventX(), fltk.EventY())
	case fltk.LEAVE:
		w.mapDrawer.Leave()
	}
	return false
}
