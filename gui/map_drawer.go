package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"sync"
	"time"

	"github.com/golang/geo/r2"
	"github.com/golang/geo/s2"
	"github.com/golang/groupcache/lru"
	"github.com/pwiecz/portal_patterns/gui/gl"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
	"golang.org/x/image/draw"
)

var projection = s2.NewMercatorProjection(180)
var black = gl.ToColor(color.NRGBA{0, 0, 0, 255})
var white = gl.ToColor(color.NRGBA{255, 255, 255, 255})
var gray = gl.ToColor(color.NRGBA{128, 128, 128, 255})
var purple = gl.ToColor(color.NRGBA{100, 50, 225, 175})
var transparent = gl.ToColor(color.NRGBA{0, 0, 0, 0})

const fontSize = 18

type mapPortal struct {
	latLng      s2.LatLng
	coords      r2.Point
	fillColor   gl.Color
	strokeColor gl.Color
	name        string
	guid        string
	drawOrder   int
}

type lockedCoordSet struct {
	set   map[osm.TileCoord]struct{}
	mutex sync.Mutex
}

func (l *lockedCoordSet) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.set = make(map[osm.TileCoord]struct{})
}
func (l *lockedCoordSet) Contains(coord osm.TileCoord) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if _, ok := l.set[coord]; ok {
		return true
	}
	return false
}
func (l *lockedCoordSet) Insert(coord osm.TileCoord) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.set[coord] = struct{}{}
}
func (l *lockedCoordSet) Remove(coord osm.TileCoord) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	delete(l.set, coord)
}
func (l *lockedCoordSet) Set(set map[osm.TileCoord]struct{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for coord := range l.set {
		if _, ok := set[coord]; !ok {
			delete(l.set, coord)
		}
	}
	for coord := range set {
		l.set[coord] = struct{}{}
	}
}
func newLockedCoordSet() *lockedCoordSet {
	return &lockedCoordSet{
		set: make(map[osm.TileCoord]struct{}),
	}
}

type lockedTileCache struct {
	cache *lru.Cache
	mutex sync.Mutex
}

func newLockedTileCache(capacity int) *lockedTileCache {
	return &lockedTileCache{
		cache: lru.New(capacity),
	}
}
func (c *lockedTileCache) Get(coord osm.TileCoord) image.Image {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	img, ok := c.cache.Get(coord)
	if !ok {
		return nil
	}
	return img.(image.Image)
}
func (c *lockedTileCache) Add(coord osm.TileCoord, img image.Image) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache.Add(coord, img)
}

type MapDrawer struct {
	renderer                   *gl.GLRenderer
	initialized                bool
	tileCache                  *lockedTileCache
	mapTiles                   map[osm.TileCoord]uint32
	missingTiles               *lockedCoordSet
	portals                    []mapPortal
	portalIndex                *PortalIndex
	paths                      [][]r2.Point
	portalIndices              map[string]int
	portalDrawOrder            []int
	defaultPortalColor         gl.Color
	taskQueue                  TaskQueue
	tileFetcher                *osm.MapTiles
	width, height              float32
	zoom                       int
	zoomPow                    float64
	x0, y0                     float64
	selectionMode              SelectionMode
	selX0, selY0, selX1, selY1 float32
	mouseX, mouseY             int
	portalUnderMouse           int
	tooltip                    string
	tooltipX, tooltipY         float32
	onMapChangedCallbacks      []func()
}

func NewMapDrawer(width, height int, tileFetcher *osm.MapTiles) *MapDrawer {
	w := &MapDrawer{
		tileCache:          newLockedTileCache(1000),
		mapTiles:           make(map[osm.TileCoord]uint32),
		missingTiles:       newLockedCoordSet(),
		defaultPortalColor: gl.ToColor(color.NRGBA{255, 127, 0, 127}),
		tileFetcher:        tileFetcher,
		portalUnderMouse:   -1,
		portalIndices:      make(map[string]int),
		width:              float32(width),
		height:             float32(height),
	}
	return w
}

func (w *MapDrawer) Destroy() {
	if w.renderer != nil {
		for _, texture := range w.mapTiles {
			gl.DeleteTexture(texture)
		}
		w.renderer.Dispose()
		w.renderer = nil
	}
}

func (w *MapDrawer) Async(callback func()) {
	w.taskQueue.Enqueue(callback)
}
func (w *MapDrawer) SetPortalColor(guid string, fillColor, strokeColor color.Color) {
	w.Async(func() {
		w.portals[w.portalIndices[guid]].fillColor = gl.ToColor(fillColor)
		w.portals[w.portalIndices[guid]].strokeColor = gl.ToColor(strokeColor)
	})
	w.MapChanged()
}

func (w *MapDrawer) Lower(guid string) {
	w.Async(func() {
		loweredPortalIndex := w.portalIndices[guid]
		drawOrder := w.portals[loweredPortalIndex].drawOrder
		if drawOrder == 0 {
			return
		}
		for i := len(w.portals) - 1; i >= 0; i-- {
			if i == 0 {
				w.portalDrawOrder[i] = loweredPortalIndex
			} else if i <= drawOrder {
				w.portalDrawOrder[i] = w.portalDrawOrder[i-1]
			}
		}
		for ord, portalIndex := range w.portalDrawOrder {
			w.portals[portalIndex].drawOrder = ord
		}
	})
	w.MapChanged()
}
func (w *MapDrawer) Raise(guid string) {
	w.Async(func() {
		raisedPortalIndex := w.portalIndices[guid]
		drawOrder := w.portals[raisedPortalIndex].drawOrder
		if drawOrder == len(w.portals)-1 {
			return
		}
		for i := 0; i < len(w.portals); i++ {
			if i == len(w.portals)-1 {
				w.portalDrawOrder[i] = raisedPortalIndex
			} else if i >= drawOrder {
				w.portalDrawOrder[i] = w.portalDrawOrder[i+1]
			}
		}
		for ord, portalIndex := range w.portalDrawOrder {
			w.portals[portalIndex].drawOrder = ord
		}
	})
	w.MapChanged()
}
func (w *MapDrawer) Resize(width, height int) {
	w.Async(func() {
		if w.width == float32(width) && w.height == float32(height) {
			return
		}
		w.width = float32(width)
		w.height = float32(height)
		w.redrawTiles()
	})
	w.MapChanged()
}
func (w *MapDrawer) Drag(dx, dy int) {
	w.Async(func() {
		if dx == 0 && dy == 0 {
			return
		}
		w.x0 += float64(dx)
		w.y0 += float64(dy)
		w.redrawTiles()
	})
	w.MapChanged()
}
func minMax(v0, v1 int) (float32, float32) {
	if v0 < v1 {
		return float32(v0), float32(v1)
	}
	return float32(v1), float32(v0)
}
func (w *MapDrawer) ShowRectangularSelection(x0, y0, x1, y1 int) {
	w.Async(func() {
		w.selX0, w.selX1 = minMax(x0, x1)
		w.selY0, w.selY1 = minMax(y0, y1)
	})
	w.MapChanged()
}
func (w *MapDrawer) PortalsInsideSelection() map[string]struct{} {
	p0 := w.screenPointToGeoPoint(int(w.selX0), int(w.selY0))
	p1 := w.screenPointToGeoPoint(int(w.selX1), int(w.selY1))
	rect := s2.RectFromLatLng(s2.LatLngFromPoint(p0))
	rect = rect.AddPoint(s2.LatLngFromPoint(p1))
	portals := make(map[string]struct{})
	for _, portal := range w.portals {
		if rect.ContainsPoint(s2.PointFromLatLng(portal.latLng)) {
			portals[portal.guid] = struct{}{}
		}
	}
	return portals

}
func (w *MapDrawer) ZoomIn(x, y int) {
	w.Async(func() {
		if w.zoom < osm.MAX_ZOOM_LEVEL {
			w.zoom++
			w.zoomPow *= 2.0
			w.x0 = (w.x0+float64(x))*2.0 - float64(x)
			w.y0 = (w.y0+float64(y))*2.0 - float64(y)
			w.redrawTiles()
		}
	})
	w.MapChanged()
}
func (w *MapDrawer) ZoomOut(x, y int) {
	w.Async(func() {
		if w.zoom > 0 {
			w.zoom--
			w.zoomPow /= 2.0
			w.x0 = (w.x0+float64(x))*0.5 - float64(x)
			w.y0 = (w.y0+float64(y))*0.5 - float64(y)
			w.redrawTiles()
		}
	})
	w.MapChanged()
}
func (w *MapDrawer) ScrollUp() {
	w.Async(func() {
		w.y0 -= 10
		w.redrawTiles()
	})
	w.MapChanged()
}
func (w *MapDrawer) ScrollDown() {
	w.Async(func() {
		w.y0 += 10
		w.redrawTiles()
	})
	w.MapChanged()
}
func (w *MapDrawer) ScrollLeft() {
	w.Async(func() {
		w.x0 -= 10
		w.redrawTiles()
	})
	w.MapChanged()
}
func (w *MapDrawer) ScrollRight() {
	w.Async(func() {
		w.x0 += 10
		w.redrawTiles()
	})
	w.MapChanged()
}

func (w *MapDrawer) SetSelectionMode(selectionMode SelectionMode) {
	w.Async(func() {
		w.selectionMode = selectionMode
	})
	w.MapChanged()
}
func (w *MapDrawer) screenPointToGeoPoint(x, y int) s2.Point {
	mapX := (float64(x) + w.x0) / 256 / w.zoomPow
	mapY := (float64(y) + w.y0) / 256 / w.zoomPow
	projectedX := mapX*360 - 180
	projectedY := 180 - mapY*360
	return projection.Unproject(r2.Point{X: projectedX, Y: projectedY})
}

func (w *MapDrawer) Hover(x, y int) {
	w.Async(func() {
		if x >= 20 && x < 60 && y >= 20 && y < 60 {
			if w.portalUnderMouse != -1 {
				w.portalUnderMouse = 1
			}
			w.tooltip = "Rectangular selection"
			w.tooltipX, w.tooltipY = 65, 40
			return
		}
		w.tooltip = ""
		w.mouseX, w.mouseY = x, y
		if len(w.portals) == 0 {
			return
		}
		portalIx, ok := w.portalIndex.ClosestPortal(w.screenPointToGeoPoint(x, y))
		if !ok {
			return
		}
		closestPortal := w.portals[portalIx]
		mapX := (float64(x) + w.x0) / 256 / w.zoomPow
		mapY := (float64(y) + w.y0) / 256 / w.zoomPow
		dx, dy := mapX-closestPortal.coords.X, mapY-closestPortal.coords.Y
		dx, dy = dx*256*w.zoomPow, dy*256*w.zoomPow
		portalUnderMouse := -1
		if dx*dx+dy*dy <= gl.PortalCircleRadius*gl.PortalCircleRadius {
			portalUnderMouse = portalIx
		}
		if portalUnderMouse != w.portalUnderMouse {
			w.portalUnderMouse = portalUnderMouse
		}
	})
	w.MapChanged()
}
func (w *MapDrawer) Leave() {
	w.Async(func() {
		w.portalUnderMouse = -1
	})
	w.MapChanged()
}
func (w *MapDrawer) ScrollToPortal(guid string) {
	w.Async(func() {
		portalCoords := w.portals[w.portalIndices[guid]].coords
		x := portalCoords.X*w.zoomPow*256 - w.x0
		y := portalCoords.Y*w.zoomPow*256 - w.y0
		if x >= 0 && x < float64(w.width) && y >= 0 && y < float64(w.height) {
			return
		}
		w.x0 = portalCoords.X*w.zoomPow*256 - float64(w.width*0.5)
		w.y0 = portalCoords.Y*w.zoomPow*256 - float64(w.height*0.5)
		w.redrawTiles()
	})
	w.MapChanged()
}

func (w *MapDrawer) ResetView() {
	w.Async(func() {
		if len(w.portals) == 0 {
			w.zoom = 0
			w.zoomPow = 1
			w.x0 = 0
			w.y0 = 0
			w.portalUnderMouse = -1
			w.redrawTiles()
			return
		}
		minX, minY, maxX, maxY := math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
		for _, portal := range w.portals {
			minX = math.Min(portal.coords.X, minX)
			minY = math.Min(portal.coords.Y, minY)
			maxX = math.Max(portal.coords.X, maxX)
			maxY = math.Max(portal.coords.Y, maxY)
		}
		numTilesX := math.Ceil(float64(w.width) / 256)
		numTilesY := math.Ceil(float64(w.height) / 256)
		for w.zoom = 19; w.zoom >= 0; w.zoom-- {
			zoomPow := math.Pow(2., float64(w.zoom))
			minXTile, minYTile := math.Floor(minX*zoomPow), math.Floor(minY*zoomPow)
			maxXTile, maxYTile := math.Floor(maxX*zoomPow), math.Floor(maxY*zoomPow)
			if maxXTile-minXTile+1 <= numTilesX && maxYTile-minYTile+1 <= numTilesY {
				break
			}
		}
		if w.zoom < 0 {
			w.zoom = 0
		}
		w.zoomPow = math.Pow(2., float64(w.zoom))
		w.x0 = (maxX+minX)*0.5*w.zoomPow*256 - float64(w.width)*0.5
		w.y0 = (maxY+minY)*0.5*w.zoomPow*256 - float64(w.height)*0.5
		w.portalUnderMouse = -1
		w.redrawTiles()
	})
	w.MapChanged()
}

func (w *MapDrawer) OnMapChanged(callback func()) {
	w.onMapChangedCallbacks = append(w.onMapChangedCallbacks, callback)
}
func (w *MapDrawer) MapChanged() {
	for _, callback := range w.onMapChangedCallbacks {
		callback()
	}
}
func (w *MapDrawer) Init(screenWidth, screenHeight int) {
	if w.initialized {
		return
	}
	renderer, err := gl.NewGLRenderer()
	if err != nil {
		panic(err)
	}

	w.renderer = renderer

	w.onNewPortals(nil)
	w.initialized = true
}
func (w *MapDrawer) Update() {
	if !w.taskQueue.Empty() {
		w.renderer.Clear(w.width, w.height)
		for !w.taskQueue.Empty() {
			callback := w.taskQueue.Dequeue()
			callback()
		}
		w.drawAllTiles()
		w.drawAllPortals()
		w.drawAllPaths()
		w.drawPortalLabel()
		w.drawTooltip()
		w.drawSelection()
		w.drawSelectionButton()
		w.drawCopyrightLabel()
	}
	w.renderer.Render(w.width, w.height)
}

func (w *MapDrawer) onNewPortals(portals []lib.Portal) {
	w.portals = make([]mapPortal, 0, len(portals))
	w.portalIndex = NewPortalIndex(portals)
	w.portalIndices = map[string]int{}
	w.portalDrawOrder = w.portalDrawOrder[:0]
	if len(portals) == 0 {
		w.zoom = 0
		w.zoomPow = 1
		w.x0 = 0
		w.y0 = 0
		w.portalUnderMouse = -1
		w.redrawTiles()
		return
	}
	for i, portal := range portals {
		mapCoords := projection.FromLatLng(portal.LatLng)
		mapCoords.X = (mapCoords.X + 180) / 360
		mapCoords.Y = (180 - mapCoords.Y) / 360
		w.portals = append(w.portals, mapPortal{
			latLng:      portal.LatLng,
			coords:      mapCoords,
			fillColor:   w.defaultPortalColor,
			strokeColor: transparent,
			name:        portal.Name,
			guid:        portal.Guid,
			drawOrder:   i,
		})
		w.portalIndices[portal.Guid] = i
		w.portalDrawOrder = append(w.portalDrawOrder, i)
	}
	w.portalUnderMouse = -1
	w.ResetView()
	w.redrawTiles()
}

const LabelXMargin = 10

func (w *MapDrawer) drawCopyrightLabel() {
	label := "© OpenStreetMap"
	textSizeX, textSizeY := w.renderer.CalculateTextSize(label, fontSize)
	posX, posY := w.width-textSizeX-5, w.height-textSizeY-5
	w.renderer.AddText(posX, posY, fontSize, black, label)
}

func (w *MapDrawer) drawPortalLabel() {
	if w.portalUnderMouse < 0 || w.portalUnderMouse >= len(w.portals) {
		return
	}
	portal := w.portals[w.portalUnderMouse]
	x := float32(portal.coords.X*w.zoomPow*256 - w.x0)
	y := float32(portal.coords.Y*w.zoomPow*256 - w.y0)
	textSizeX, textSizeY := w.renderer.CalculateTextSize(portal.name, fontSize)
	labelPosX, labelPosY := x-textSizeX/2-LabelXMargin, y-gl.PortalCircleRadius-2
	if labelPosX < 0 {
		labelPosX = 0
	} else if labelPosX+textSizeX >= w.width {
		labelPosX = w.width - textSizeX - LabelXMargin*2
	}
	if labelPosY-textSizeY < 0 {
		labelPosY = y + 5 + textSizeY + 4
	}
	w.renderer.AddRectFilled(labelPosX, labelPosY-textSizeY, labelPosX+textSizeX+2*LabelXMargin, labelPosY, white)
	textPosX := labelPosX + LabelXMargin
	w.renderer.AddText(textPosX, labelPosY-textSizeY, fontSize, black, portal.name)
}
func (w *MapDrawer) drawTooltip() {
	if w.tooltip == "" {
		return
	}
	textSizeX, textSizeY := w.renderer.CalculateTextSize(w.tooltip, fontSize)
	width, height := textSizeX+10, textSizeY+6
	tooltipY := w.tooltipY - height/2
	w.renderer.AddRectFilled(w.tooltipX, tooltipY, w.tooltipX+width, tooltipY+height, gray)
	w.renderer.AddText(w.tooltipX+5, tooltipY+3, fontSize, black, w.tooltip)
}
func (w *MapDrawer) drawAllTiles() {
	for coord, tex := range w.mapTiles {
		dx := float32(coord.X)*256 - float32(w.x0)
		dy := float32(coord.Y)*256 - float32(w.y0)
		w.renderer.AddImage(uint32(tex), dx, dy, dx+256, dy+256)
	}
}
func (w *MapDrawer) drawAllPortals() {
	for _, portalIndex := range w.portalDrawOrder {
		portal := w.portals[portalIndex]
		x := float32(portal.coords.X*w.zoomPow*256 - w.x0)
		y := float32(portal.coords.Y*w.zoomPow*256 - w.y0)
		w.renderer.AddCircleFilled(x, y, portal.fillColor)
		w.renderer.AddCircle(x, y, portal.strokeColor)
	}
}
func (w *MapDrawer) drawAllPaths() {
	for _, path := range w.paths {
		for i := 1; i < len(path); i++ {
			x0 := float32(path[i-1].X*w.zoomPow*256 - w.x0)
			y0 := float32(path[i-1].Y*w.zoomPow*256 - w.y0)
			x1 := float32(path[i].X*w.zoomPow*256 - w.x0)
			y1 := float32(path[i].Y*w.zoomPow*256 - w.y0)
			w.renderer.AddLine(x0, y0, x1, y1, 3, purple)
		}
	}
}
func (w *MapDrawer) drawSelectionButton() {

	if w.selectionMode == RectangularSelection {
		w.renderer.AddSelectionButton(20, 20, black, white)
	} else {
		w.renderer.AddSelectionButton(20, 20, white, black)
	}
}
func (w *MapDrawer) drawSelection() {
	if w.selX0 >= w.selX1 || w.selY0 >= w.selY1 {
		return
	}
	w.renderer.AddRect(w.selX0, w.selY0, w.selX1, w.selY1, 1, black)
}
func (w *MapDrawer) SetPortals(portals []lib.Portal) {
	w.Async(func() { w.onNewPortals(portals) })
	w.MapChanged()
}
func (w *MapDrawer) SetPaths(paths [][]s2.Point) {
	w.Async(func() {
		tesselator := s2.NewEdgeTessellator(projection, 1e-3)
		w.paths = w.paths[:0]
		for _, path := range paths {
			mapPath := []r2.Point{}
			for i := 1; i < len(path); i++ {
				mapPath = tesselator.AppendProjected(path[i-1], path[i], mapPath)
			}
			for i := range mapPath {
				mapPath[i].X = (mapPath[i].X + 180) / 360
				mapPath[i].Y = (180 - mapPath[i].Y) / 360
			}
			w.paths = append(w.paths, mapPath)
		}
	})
	w.MapChanged()
}
func (w *MapDrawer) onTileRead(coord osm.TileCoord, img image.Image) {
	wrappedCoord := coord
	maxCoord := 1 << coord.Zoom
	for wrappedCoord.X < 0 {
		wrappedCoord.X += maxCoord
	}
	wrappedCoord.X %= maxCoord
	w.tileCache.Add(wrappedCoord, img)
	w.missingTiles.Remove(coord)
	w.Async(func() { w.showTile(coord, img) })
	w.MapChanged()
}
func (w *MapDrawer) redrawTiles() {
	if w.zoomPow == 0 {
		return
	}
	tileCoords := make(map[osm.TileCoord]struct{})
	maxCoord := 1 << w.zoom
	x1, y1 := w.x0+float64(w.width), w.y0+float64(w.height)
	for x := int(math.Floor(w.x0 / 256)); x <= int(math.Floor(x1/256)); x++ {
		for y := int(math.Floor(w.y0 / 256)); y <= int(math.Floor(y1/256)); y++ {
			if y >= 0 && y < maxCoord {
				tileCoords[osm.TileCoord{X: x, Y: y, Zoom: w.zoom}] = struct{}{}
			}
		}
	}
	w.tileFetcher.CancelRequestsExcept(tileCoords)
	for coord, tex := range w.mapTiles {
		if _, ok := tileCoords[coord]; !ok {
			gl.DeleteTexture(tex)
			delete(w.mapTiles, coord)
		} else {
			delete(tileCoords, coord)
		}
	}
	w.missingTiles.Set(tileCoords)
	for coord := range tileCoords {
		w.tryShowTile(coord)
	}
}
func (w *MapDrawer) showTile(coord osm.TileCoord, img image.Image) {
	if tex, ok := w.mapTiles[coord]; ok {
		gl.DeleteTexture(tex)
	}
	w.mapTiles[coord] = gl.NewTexture(img)
	w.MapChanged()
}

func (w *MapDrawer) fetchTile(coord osm.TileCoord) {
	for {
		img, err := w.tileFetcher.GetTile(coord)
		if err == nil {
			w.onTileRead(coord, img)
			return
		}
		if !errors.Is(err, osm.ErrBusy) && !errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "fetching error:", err)
			return
		}
		if !w.missingTiles.Contains(coord) {
			return
		}
		// try fetching again after 1 second
		timer := time.NewTimer(time.Second)
		<-timer.C
		// check if we still need the tile before refetching
		if !w.missingTiles.Contains(coord) {
			return
		}
	}
}

func (w *MapDrawer) tryShowTile(coord osm.TileCoord) {
	wrappedCoord := coord
	maxCoord := 1 << coord.Zoom
	for wrappedCoord.X < 0 {
		wrappedCoord.X += maxCoord
	}
	wrappedCoord.X %= maxCoord
	tileImage := w.tileCache.Get(wrappedCoord)
	if tileImage != nil {
		w.missingTiles.Remove(coord)
	} else {
		go func() {
			w.fetchTile(coord)
		}()
		if wrappedCoord.Zoom > 0 {
			w.missingTiles.Insert(coord)
			zoomedOutCoord := osm.TileCoord{X: wrappedCoord.X / 2, Y: wrappedCoord.Y / 2, Zoom: wrappedCoord.Zoom - 1}
			if zoomedOutTileImage := w.tileCache.Get(zoomedOutCoord); zoomedOutTileImage != nil {
				sourceX := (wrappedCoord.X % 2) * 128
				sourceY := (wrappedCoord.Y % 2) * 128

				img := image.NewRGBA(zoomedOutTileImage.Bounds())
				draw.NearestNeighbor.Scale(img, img.Bounds(), zoomedOutTileImage, image.Rect(sourceX, sourceY, sourceX+128, sourceY+128), draw.Over, nil)
				tileImage = img
			}
		}
	}
	if tileImage != nil {
		w.showTile(coord, tileImage)
	}
}
