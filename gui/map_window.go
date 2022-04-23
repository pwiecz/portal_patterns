package main

import (
	"image/color"
	"log"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

type MapWindow struct {
	*fltk.GlWindow
	parent                   *fltk.Window
	mapDrawer                *MapDrawer
	isMouseIn                bool
	prevX, prevY             int
	selectionChangedCallback func(map[string]struct{})
	addedToSelectionCallback func(map[string]struct{})
	windowClosedCallback     func()
	rightClickCallback       func(string, int, int)
	selectionMode            SelectionMode
	firstShow                bool
}

func NewMapWindow(title string, tileFetcher *osm.MapTiles, parent *fltk.Window) *MapWindow {
	w := &MapWindow{}
	w.GlWindow = fltk.NewGlWindow(0, 0, 900, 870, w.drawMap)
	w.GlWindow.SetEventHandler(w.handleEvent)
	w.GlWindow.SetResizeHandler(w.onGlWindowResized)
	w.GlWindow.SetMode(fltk.ALPHA | fltk.DOUBLE | fltk.MULTISAMPLE | fltk.OPENGL3)
	w.parent = parent
	w.mapDrawer = NewMapDrawer(900, 870, tileFetcher)
	w.mapDrawer.OnMapChanged(w.redraw)
	w.firstShow = true
	w.Resizable(w.GlWindow)
	return w
}

func (w *MapWindow) redraw() {
	fltk.Awake(w.Redraw)
}
func (w *MapWindow) Destroy() {
	w.mapDrawer.Destroy()
	w.GlWindow.Destroy()
}

type SelectionMode int

var NoSelection SelectionMode = 0
var RectangularSelection SelectionMode = 1

func (w *MapWindow) SetSelectionMode(selectionMode SelectionMode) {
	w.selectionMode = selectionMode
	w.mapDrawer.SetSelectionMode(selectionMode)
	w.setCursor()
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
func (w *MapWindow) setCursor() {
	if !w.isMouseIn {
		w.parent.SetCursor(fltk.CURSOR_DEFAULT)
		return
	}
	switch w.selectionMode {
	case NoSelection:
		w.parent.SetCursor(fltk.CURSOR_ARROW)
	case RectangularSelection:
		w.parent.SetCursor(fltk.CURSOR_CROSS)
	}
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
		if w.selectionMode == NoSelection && fltk.EventButton() == fltk.LeftMouse && fltk.EventIsClick() {
			x, y := fltk.EventX(), fltk.EventY()
			if x >= 20 && x < 60 && y >= 20 && y < 60 {
				w.SetSelectionMode(RectangularSelection)
				return true
			}
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
		} else if w.selectionMode == RectangularSelection {
			selection := w.mapDrawer.PortalsInsideSelection()
			w.mapDrawer.ShowRectangularSelection(0, 0, 0, 0)
			if fltk.EventState()&fltk.CTRL != 0 {
				if w.addedToSelectionCallback != nil {
					w.addedToSelectionCallback(selection)
				}
			} else {
				if w.selectionChangedCallback != nil {
					w.selectionChangedCallback(selection)
				}
			}
			w.SetSelectionMode(NoSelection)
			return true
		}
	case fltk.DRAG:
		if fltk.EventButton1() {
			currX, currY := fltk.EventX(), fltk.EventY()
			switch w.selectionMode {
			case NoSelection:
				w.mapDrawer.Drag(w.prevX-currX, w.prevY-currY)
				w.prevX, w.prevY = currX, currY
				w.redraw()
				return true
			case RectangularSelection:
				w.mapDrawer.ShowRectangularSelection(w.prevX, w.prevY, currX, currY)
				w.redraw()
				return true
			}
		}
	case fltk.MOUSEWHEEL:
		// For some reason on Windows that's the most precise way
		// to get the actual mouse position for this event.
		x, y := fltk.EventXRoot()-w.XRoot(), fltk.EventYRoot()-w.YRoot()
		dy := fltk.EventDY()
		if dy < 0 {
			w.mapDrawer.ZoomIn(x, y)
			w.redraw()
			return true
		} else if dy > 0 {
			w.mapDrawer.ZoomOut(x, y)
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
		w.isMouseIn = true
		w.setCursor()
		w.mapDrawer.Hover(fltk.EventX(), fltk.EventY())
	case fltk.ENTER:
		w.isMouseIn = true
		w.setCursor()
	case fltk.LEAVE:
		w.mapDrawer.Leave()
		w.isMouseIn = false
		w.setCursor()
	case fltk.SHOW:
		if w.firstShow && w.IsShown() {
			w.firstShow = false
			w.MakeCurrent()
			if err := gl.Init(); err != nil {
				log.Fatal("Cannot initialize OpenGL", err)
			}
			w.redraw()
		}
	}
	return false
}

func (w *MapWindow) onGlWindowResized() {
	if w.mapDrawer != nil {
		w.mapDrawer.Resize(w.W(), w.H())
	}
}
