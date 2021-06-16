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
	*fltk.GlWindow
	mapDrawer                *MapDrawer
	prevX, prevY             int
	selectionChangedCallback func(map[string]struct{})
	addedToSelectionCallback func(map[string]struct{})
	windowClosedCallback     func()
	rightClickCallback       func(string, int, int)
}

func NewMapWindow(title string, tileFetcher *osm.MapTiles) *MapWindow {
	w := &MapWindow{}
	w.GlWindow = fltk.NewGlWindow(0, 0, 900, 870, w.drawMap)
	w.GlWindow.SetEventHandler(w.handleEvent)
	w.GlWindow.SetResizeHandler(w.onGlWindowResized)
	w.mapDrawer = NewMapDrawer(900, 870, tileFetcher)
	w.mapDrawer.OnMapChanged(w.redraw)
	w.Resizable(w.GlWindow)
	return w
}

func (w *MapWindow) redraw() {
	if (fltk.Lock()) {
		defer fltk.Unlock()
		w.Redraw()
	}
}
func (w *MapWindow) Destroy() {
	w.mapDrawer.Destroy()
	w.GlWindow.Destroy()
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
func (w *MapWindow) SetPortals(portals []lib.Portal) {
	w.mapDrawer.SetPortals(portals)
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
func (w *MapWindow) SetPortalColor(guid string, fillColor, strokeColor color.Color) {
	w.mapDrawer.SetPortalColor(guid, fillColor, strokeColor)
}
func (w *MapWindow) ScrollToPortal(guid string) {
	w.mapDrawer.ScrollToPortal(guid)
}
func (w *MapWindow) drawMap() {
	if !w.Valid() {
		if err := gl.Init(); err != nil {
			log.Fatal("Cannot initialize OpenGL", err)
		}
	}
	if !w.ContextValid() {
		_, _, width, height := fltk.ScreenWorkArea(0 /* main screen */)
		w.mapDrawer.Init(width, height)
	}

	w.mapDrawer.Update()
}

func (w *MapWindow) ZoomIn() {
	w.mapDrawer.ZoomIn(int(w.mapDrawer.width/2), int(w.mapDrawer.height/2))
}
func (w *MapWindow) ZoomOut() {
	w.mapDrawer.ZoomOut(int(w.mapDrawer.width/2), int(w.mapDrawer.height/2))
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
			w.redraw()
			return true
		}
	case fltk.MOUSEWHEEL:
		dy := fltk.EventDY()
		if dy < 0 {
			w.mapDrawer.ZoomIn(fltk.EventX(), fltk.EventY())
			w.redraw()
			return true
		} else if dy > 0 {
			w.mapDrawer.ZoomOut(fltk.EventX(), fltk.EventY())
			w.redraw()
			return true
		}
	case fltk.KEY:
		if (fltk.EventState()&fltk.CTRL) != 0 &&
			(fltk.EventKey() == '+' || fltk.EventKey() == '=') {
			w.mapDrawer.ZoomIn(w.W()/2, w.H()/2)
			w.redraw()
			return true
		} else if fltk.EventKey() == '-' && (fltk.EventState()&fltk.CTRL) != 0 {
			w.mapDrawer.ZoomOut(w.W()/2, w.H()/2)
			w.redraw()
			return true
		}
	case fltk.MOVE:
		w.mapDrawer.Hover(fltk.EventX(), fltk.EventY())
	case fltk.LEAVE:
		w.mapDrawer.Leave()
	}
	return false
}

func (w *MapWindow) onGlWindowResized() {
	if w.mapDrawer != nil {
		w.mapDrawer.Resize(w.W(), w.H())
	}
}
