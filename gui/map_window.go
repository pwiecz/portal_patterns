package main

import (
	"image/color"
	"log"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type MapWindow struct {
	window                   *fltk.Window
	glWindow                 *fltk.GlWindow
	mapDrawer                *MapDrawer
	prevX, prevY             int
	selectionChangedCallback func(map[string]struct{})
	addedToSelectionCallback func(map[string]struct{})
	windowClosedCallback     func()
	rightClickCallback       func(string, int, int)
}

func NewMapWindow(title string, tileFetcher *osm.MapTiles) *MapWindow {
	w := &MapWindow{}
	w.window = fltk.NewWindow(800, 600)
	w.window.SetLabel(title + " - © OpenStreetMap")
	w.window.Begin()
	w.window.SetCallback(w.onWindowClosed)
	w.glWindow = fltk.NewGlWindow(0, 0, 800, 600, w.drawMap)
	w.glWindow.SetEventHandler(w.handleEvent)
	w.glWindow.SetResizeHandler(w.onGlWindowResized)
	w.window.End()
	w.window.Resizable(w.glWindow)
	w.window.Show()
	w.mapDrawer = NewMapDrawer(800, 600, tileFetcher)
	w.mapDrawer.OnMapChanged(
		func() {
			fltk.Awake(w.glWindow.Redraw)
		})
	return w
}

func (w *MapWindow) Destroy() {
	w.glWindow.Destroy()
	w.window.Destroy()
}
func (w *MapWindow) SetSelectionChangeCallback(callback func(map[string]struct{})) {
	w.selectionChangedCallback = callback
}
func (w *MapWindow) SetAddedToSelectionCallback(callback func(map[string]struct{})) {
	w.addedToSelectionCallback = callback
}
func (w *MapWindow) SetRightClickCallback(callback func(string, int, int)) {
	w.rightClickCallback = callback
}
func (w *MapWindow) SetWindowClosedCallback(callback func()) {
	w.windowClosedCallback = callback
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
func portalsToPoints(portals []lib.Portal) []s2.Point {
	points := make([]s2.Point, 0, len(portals))
	for _, portal := range portals {
		points = append(points, s2.PointFromLatLng(portal.LatLng))
	}
	return points

}
func (w *MapWindow) SetPortalPaths(portalPaths [][]lib.Portal) {
	paths := make([][]s2.Point, 0, len(portalPaths))
	for _, portalPath := range portalPaths {
		paths = append(paths, portalsToPoints(portalPath))
	}
	w.SetPaths(paths)
}
func (w *MapWindow) SetPaths(paths [][]s2.Point) {
	w.mapDrawer.SetPaths(paths)
}
func (w *MapWindow) Raise(guid string) {
	w.mapDrawer.Raise(guid)
}
func (w *MapWindow) Lower(guid string) {
	w.mapDrawer.Lower(guid)
}
func (w *MapWindow) SetPortalColor(guid string, color color.Color) {
	w.mapDrawer.SetPortalColor(guid, color)
}
func (w *MapWindow) drawMap() {
	if !w.glWindow.Valid() {
		if err := gl.Init(); err != nil {
			log.Fatal("Cannot initialize OpenGL", err)
		}
	}
	if !w.glWindow.ContextValid() {
		_, _, width, height := fltk.ScreenWorkArea(0 /* main screen */)
		w.mapDrawer.Init(width, height)
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
	case fltk.RELEASE:
		if fltk.EventButton() == fltk.LeftMouse && fltk.EventIsClick() {
			if w.mapDrawer.portalUnderMouse >= 0 {
				selection := make(map[string]struct{})
				selection[w.mapDrawer.portals[w.mapDrawer.portalUnderMouse].guid] = struct{}{}
				if fltk.EventState()&fltk.CTRL != 0 {
					if w.addedToSelectionCallback != nil {
						w.addedToSelectionCallback(selection)
					}
				} else {
					if w.selectionChangedCallback != nil {
						w.selectionChangedCallback(selection)
					}
				}
			} else {
				if w.selectionChangedCallback != nil {
					w.selectionChangedCallback(make(map[string]struct{}))
				}
			}
			return true
		} else if fltk.EventButton() == fltk.RightMouse && fltk.EventIsClick() {
			if w.rightClickCallback != nil {
				if w.mapDrawer.portalUnderMouse >= 0 {
					portalUnderMouse := w.mapDrawer.portals[w.mapDrawer.portalUnderMouse].guid
					w.rightClickCallback(portalUnderMouse, fltk.EventX(), fltk.EventY())
				} else {
					w.rightClickCallback("", fltk.EventX(), fltk.EventY())
				}
			}
		}
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

func (w *MapWindow) onWindowClosed() {
	if w.windowClosedCallback != nil {
		w.windowClosedCallback()
	}
}
func (w *MapWindow) onGlWindowResized() {
	if w.mapDrawer != nil {
		w.mapDrawer.Resize(w.glWindow.W(), w.glWindow.H())
	}
}
