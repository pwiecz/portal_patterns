package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	math "github.com/chewxy/math32"
	"golang.org/x/image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/golang/geo/r2"
	"github.com/golang/geo/s2"
	"github.com/golang/groupcache/lru"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
)

const portalCircleRadius = 13

var projection = s2.NewMercatorProjection(180)

type mapPortal struct {
	GUID        string
	Coords      r2.Point
	Name        string
	Color       color.NRGBA
	StrokeColor color.NRGBA
}

type SolutionMap struct {
	widget.BaseWidget
	//	*tk.Window
	//	font               *tk.SysFont
	//	fontDescription    string
	//	textAscent         int
	//	textDescent        int
	//	layout             *tk.PackLayout
	//	canvas             *tk.Canvas
	//	nameLabel          *tk.CanvasText
	//	nameLabelBackgound *tk.CanvasRectangle
	zoom          int
	x0, y0        float32
	width, height float32
	// workaround for Fyne not setting the Position in the ScrollEvent.
	lastMousePosition fyne.Position
	mouseDownPosition fyne.Position

	portalPaths *[][]r2.Point
	tileFetcher *osm.MapTiles
	mutex       sync.Mutex
	tileCache   *lru.Cache
	//	lines              []*tk.CanvasLine
	visibleTiles            map[osm.TileCoord]image.Image
	visiblePlaceholderTiles map[osm.TileCoord]image.Image
	tilesToFetch            map[osm.TileCoord]struct{}

	portals       []mapPortal
	portalIndices map[string]int
	//	objects            []fyne.CanvasObject // cache for list of displayed canvas objects
	portalShapeIndex       *s2.ShapeIndex
	shapeIndexIdToPortalId map[int32]int
	portalUnderMouse       int

	// Left click not over a portal
	OnSelectionCleared func()
	// Left click over a portal (bool is true if Control was pressed)
	OnPortalSelected func(string, bool)
	// Right click
	OnContextMenu func(float32, float32)
	// Right click over a portal
	OnPortalContextMenu func(string, float32, float32)
}

var _ fyne.Draggable = &SolutionMap{}
var _ desktop.Hoverable = &SolutionMap{}
var _ desktop.Mouseable = &SolutionMap{}

func NewSolutionMap(tileFetcher *osm.MapTiles) *SolutionMap {
	s := &SolutionMap{
		tileFetcher:             tileFetcher,
		tileCache:               lru.New(1000),
		visibleTiles:            make(map[osm.TileCoord]image.Image),
		visiblePlaceholderTiles: make(map[osm.TileCoord]image.Image),
		tilesToFetch:            make(map[osm.TileCoord]struct{}),
		portalIndices:           make(map[string]int),
		portalUnderMouse:        -1,
		portalPaths:             &[][]r2.Point{},
	}
	s.ExtendBaseWidget(s)
	return s
}

type solutionMapRenderer struct {
	background       *canvas.Rectangle
	visibleTiles     map[osm.TileCoord]*canvas.Image
	circles          []*canvas.Circle
	paths            *[][]r2.Point // paths for which the lines were created
	linesZoom        int           // zoom level at which lines were drawn
	lines            []*canvas.Line
	labelText        *canvas.Text
	portalUnderMouse int
	solutionMap      *SolutionMap
}

func (m *SolutionMap) CreateRenderer() fyne.WidgetRenderer {
	return &solutionMapRenderer{
		solutionMap:      m,
		visibleTiles:     make(map[osm.TileCoord]*canvas.Image),
		background:       canvas.NewRectangle(color.White),
		portalUnderMouse: -1,
	}
}

func (r *solutionMapRenderer) Layout(size fyne.Size) {
	r.solutionMap.width = size.Width
	r.solutionMap.height = size.Height
	r.background.Resize(size)
	r.solutionMap.updateVisibleTiles()
}
func (r *solutionMapRenderer) MinSize() fyne.Size {
	return fyne.NewSize(256, 256)
}
func (r *solutionMapRenderer) Refresh() {
	r.solutionMap.mutex.Lock()
	for coords := range r.visibleTiles {
		if _, ok := r.solutionMap.visibleTiles[coords]; !ok {
			if _, ok := r.solutionMap.visiblePlaceholderTiles[coords]; !ok {
				delete(r.visibleTiles, coords)
			}
		}
	}
	for coords, img := range r.solutionMap.visibleTiles {
		if cImg, ok := r.visibleTiles[coords]; !ok || cImg.Image != img {
			r.visibleTiles[coords] = r.createImageFromImage(img)
		}
		r.positionImageAtCoords(r.visibleTiles[coords], coords)
	}
	for coords, img := range r.solutionMap.visiblePlaceholderTiles {
		if _, ok := r.visibleTiles[coords]; !ok {
			r.visibleTiles[coords] = r.createImageFromImage(img)
		}
		r.positionImageAtCoords(r.visibleTiles[coords], coords)
	}
	for i, portal := range r.solutionMap.portals {
		if i >= len(r.circles) {
			r.circles = append(r.circles, canvas.NewCircle(portal.Color))
			r.circles[i].Resize(fyne.NewSize(portalCircleRadius, portalCircleRadius))
		}
		circle := r.circles[i]
		x, y := r.geoToScreenCoordinates(float32(portal.Coords.X), float32(portal.Coords.Y))
		colorChanged := circle.FillColor != portal.Color || circle.StrokeColor != portal.StrokeColor
		circle.FillColor = portal.Color
		circle.StrokeColor = portal.StrokeColor
		circle.StrokeWidth = 2
		circle.Move(fyne.NewPos(x-6, y-6))
		if colorChanged {
			circle.Refresh()
		}
	}
	r.circles = r.circles[:len(r.solutionMap.portals)]
	if r.paths != r.solutionMap.portalPaths {
		r.lines = []*canvas.Line{}
		r.paths = r.solutionMap.portalPaths
		r.linesZoom = r.solutionMap.zoom
		for _, path := range *r.solutionMap.portalPaths {
			for i := 1; i < len(path); i++ {
				line := canvas.NewLine(color.NRGBA{0, 0, 255, 128})
				line.StrokeWidth = 4
				x0, y0 := r.geoToScreenCoordinates(float32(path[i-1].X), float32(path[i-1].Y))
				x1, y1 := r.geoToScreenCoordinates(float32(path[i].X), float32(path[i].Y))
				line.Position1 = fyne.NewPos(x0, y0)
				line.Position2 = fyne.NewPos(x1, y1)
				r.lines = append(r.lines, line)
			}
		}
	} else {
		lineIx := 0
		for _, path := range *r.solutionMap.portalPaths {
			for i := 1; i < len(path); i++ {
				x0, y0 := r.geoToScreenCoordinates(float32(path[i-1].X), float32(path[i-1].Y))
				x1, y1 := r.geoToScreenCoordinates(float32(path[i].X), float32(path[i].Y))
				line := r.lines[lineIx]
				line.Position1 = fyne.NewPos(x0, y0)
				line.Position2 = fyne.NewPos(x1, y1)
				if r.linesZoom != r.solutionMap.zoom {
					line.Resize(fyne.NewSize(x1-x0, y1-y0))
				}
				lineIx++
			}
		}
		r.linesZoom = r.solutionMap.zoom
	}
	if r.portalUnderMouse != r.solutionMap.portalUnderMouse {
		if r.solutionMap.portalUnderMouse == -1 {
			r.labelText = nil
		} else {
			portalUnderMouse := r.solutionMap.portals[r.solutionMap.portalUnderMouse]
			r.labelText = canvas.NewText(portalUnderMouse.Name, color.Black)
			x, y := r.geoToScreenCoordinates(float32(portalUnderMouse.Coords.X), float32(portalUnderMouse.Coords.Y))
			textSize := r.labelText.MinSize()
			labelPosX, labelPosY := x-textSize.Width/2, y+portalCircleRadius-2
			if labelPosX < 0 {
				labelPosX = 0
			} else if labelPosX+textSize.Width >= r.solutionMap.width {
				labelPosX = r.solutionMap.width - textSize.Width
			}
			if labelPosY-textSize.Height < 0 {
				labelPosY = y + 5 + textSize.Height + 4
			}
			r.labelText.Move(fyne.NewPos(labelPosX, labelPosY))
			r.labelText.Refresh()
		}
		r.portalUnderMouse = r.solutionMap.portalUnderMouse
	}
	r.solutionMap.mutex.Unlock()

	// Don't call refresh on tiles, as this causes the GL texture to be recreated,
	// which is costly. Refreshing background rectangle is enough to redraw the canvas.
	r.background.Refresh()
}

func (r *solutionMapRenderer) positionImageAtCoords(img *canvas.Image, coord osm.TileCoord) {
	x := float32(coord.X)*256.0 - r.solutionMap.x0
	y := float32(coord.Y)*256.0 - r.solutionMap.y0
	position := img.Position()
	if position.X == x && position.Y == y {
		return
	}
	img.Move(fyne.NewPos(x, y))
}
func (r *solutionMapRenderer) createImageFromImage(img image.Image) *canvas.Image {
	cImg := canvas.NewImageFromImage(img)
	cImg.FillMode = canvas.ImageFillOriginal
	cImg.ScaleMode = canvas.ImageScaleFastest
	cImg.Resize(fyne.NewSize(256, 256))
	return cImg
}
func (r *solutionMapRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{r.background}
	r.solutionMap.mutex.Lock()
	defer r.solutionMap.mutex.Unlock()
	for _, image := range r.visibleTiles {
		objects = append(objects, image)
	}
	for _, circle := range r.circles {
		objects = append(objects, circle)
	}
	for _, line := range r.lines {
		objects = append(objects, line)
	}
	if r.labelText != nil {
		objects = append(objects, r.labelText)
	}
	return objects
}
func (r *solutionMapRenderer) Destroy() {}
func (m *SolutionMap) isTileMissing(coords osm.TileCoord) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, ok := m.tilesToFetch[coords]
	return ok
}
func (r *SolutionMap) removeMissingTile(coords osm.TileCoord) bool {
	wasTileMissing := false
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.tilesToFetch[coords]; ok {
		wasTileMissing = true
	}
	delete(r.tilesToFetch, coords)
	return wasTileMissing
}
func (r *SolutionMap) addToCache(coords osm.TileCoord, image image.Image) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.tileCache.Add(coords, image)
}
func (r *SolutionMap) getFromCache(coords osm.TileCoord) (image.Image, bool) {
	r.mutex.Lock()
	tile, ok := r.tileCache.Get(coords)
	r.mutex.Unlock()
	if ok {
		return tile.(image.Image), true
	}
	return nil, false
}

func (s *SolutionMap) SetPortalColor(guid string, color color.NRGBA) {
	fillColor := color
	fillColor.A = 128
	strokeColor := color
	strokeColor.A = 255
	s.mutex.Lock()
	if portalIx, ok := s.portalIndices[guid]; ok {
		s.portals[portalIx].Color = fillColor
		s.portals[portalIx].StrokeColor = strokeColor
	}
	s.mutex.Unlock()
	s.Refresh()
}

//func (s *SolutionMap) RaisePortal(guid string) {
//	if portal, ok := s.portals[guid]; ok {
//		portal.shape.Raise()
//	}
//}
func (s *SolutionMap) Clear() {
	s.mutex.Lock()
	s.zoom = 0
	s.x0 = 0
	s.y0 = 0
	s.portalPaths = &[][]r2.Point{}
	s.visibleTiles = make(map[osm.TileCoord]image.Image)
	s.visiblePlaceholderTiles = make(map[osm.TileCoord]image.Image)
	s.tilesToFetch = make(map[osm.TileCoord]struct{})
	s.portals = nil
	s.portalIndices = make(map[string]int)
	s.mutex.Unlock()
	s.Refresh()
}

func (s *SolutionMap) Dragged(event *fyne.DragEvent) {
	dx, dy := event.Dragged.DX, event.Dragged.DY
	s.x0 -= dx
	s.y0 -= dy
	s.updateVisibleTiles()
	s.Refresh()
}
func (s *SolutionMap) DragEnd()                    {}
func (s *SolutionMap) MouseIn(*desktop.MouseEvent) {}
func (s *SolutionMap) MouseMoved(event *desktop.MouseEvent) {
	s.lastMousePosition = event.Position
	if len(s.portals) == 0 {
		return
	}
	zoomPow := float64(math.Pow(2, float32(s.zoom)))
	mapX := (float64(event.Position.X) + float64(s.x0)) / 256 / zoomPow
	mapY := (float64(event.Position.Y) + float64(s.y0)) / 256 / zoomPow
	projectedX := mapX*360 - 180
	projectedY := 180 - mapY*360
	point := projection.Unproject(r2.Point{X: projectedX, Y: projectedY})
	opts := s2.NewClosestEdgeQueryOptions().MaxResults(1)
	query := s2.NewClosestEdgeQuery(s.portalShapeIndex, opts)
	target := s2.NewMinDistanceToPointTarget(point)
	result := query.FindEdges(target)
	if len(result) == 0 {
		return
	}
	shapeId := result[0].ShapeID()
	portalIx, ok := s.shapeIndexIdToPortalId[shapeId]
	if !ok {
		return
	}
	closestPortal := s.portals[portalIx]
	dx, dy := mapX-closestPortal.Coords.X, mapY-closestPortal.Coords.Y
	dx, dy = dx*256*zoomPow, dy*256*zoomPow
	portalUnderMouse := -1
	if dx*dx+dy*dy <= portalCircleRadius*portalCircleRadius {
		portalUnderMouse = portalIx
	}
	if portalUnderMouse != s.portalUnderMouse {
		s.portalUnderMouse = portalUnderMouse
		s.Refresh()
	}
}
func (s *SolutionMap) MouseOut() {}
func (s *SolutionMap) MouseDown(event *desktop.MouseEvent) {
	// Explicitely trigger MouseMove, because the mouse might have been moved without
	// the move being registered. E.g. while the context menu was shown.
	s.MouseMoved(event)
	s.mouseDownPosition = event.Position
}
func (s *SolutionMap) MouseUp(event *desktop.MouseEvent) {
	dx := s.mouseDownPosition.X - event.Position.X
	dy := s.mouseDownPosition.Y - event.Position.Y
	if dx*dx+dy*dy > 4 {
		return
	}
	if event.Button == desktop.MouseButtonPrimary {
		if s.portalUnderMouse == -1 {
			if s.OnSelectionCleared != nil {
				s.OnSelectionCleared()
			}
		} else {
			if s.OnPortalSelected != nil {
				s.OnPortalSelected(s.portals[s.portalUnderMouse].GUID,
					(event.Modifier&desktop.ControlModifier) != 0)
			}
		}
	} else if event.Button == desktop.MouseButtonSecondary {
		if s.portalUnderMouse == -1 {
			if s.OnContextMenu != nil {
				s.OnContextMenu(event.Position.X, event.Position.Y)
			}
		} else {
			if s.OnContextMenu != nil {
				s.OnPortalContextMenu(s.portals[s.portalUnderMouse].GUID,
					event.Position.X, event.Position.Y)
			}
		}
	}
}
func (s *SolutionMap) Scrolled(event *fyne.ScrollEvent) {
	if event.Scrolled.DY == 0 {
		return
	}
	x, y := s.lastMousePosition.X, s.lastMousePosition.Y
	if event.Scrolled.DY < 0 {
		if s.zoom <= 0 {
			return
		}
		s.zoom--
		s.x0 = (s.x0+x)*0.5 - x
		s.y0 = (s.y0+y)*0.5 - y
	} else {
		if s.zoom >= 19 {
			return
		}
		s.zoom++
		s.x0 = (s.x0+x)*2.0 - x
		s.y0 = (s.y0+y)*2.0 - y
	}
	s.updateVisibleTiles()
	s.Refresh()
}
func (r *SolutionMap) addTile(coords osm.TileCoord, image image.Image, placeholder bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if placeholder {
		if _, ok := r.visibleTiles[coords]; ok {
			return
		}
		if _, ok := r.visiblePlaceholderTiles[coords]; ok {
			return
		}
		r.visiblePlaceholderTiles[coords] = image
	} else {
		if _, ok := r.visibleTiles[coords]; ok {
			return
		}
		r.visibleTiles[coords] = image
		delete(r.visiblePlaceholderTiles, coords)
		delete(r.tilesToFetch, coords)
	}
}

func (r *SolutionMap) onFetchBusy(coord osm.TileCoord) {
	if !r.isTileMissing(coord) {
		return
	}
	// try fetching again after 1 second
	timer := time.NewTimer(time.Second)
	go func() {
		<-timer.C
		// check if we still need the tile before refetching
		if !r.isTileMissing(coord) {
			return
		}

		r.fetchTile(coord)
	}()
}
func (r *SolutionMap) onTileRead(coord osm.TileCoord, img image.Image) {
	if !r.removeMissingTile(coord) {
		// the tile is no longer needed
		return
	}
	r.addToCache(coord, img)
	r.addTile(coord, img, false)
	r.Refresh()
}
func (r *SolutionMap) updateVisibleTiles() {
	tileCoords := make(map[osm.TileCoord]struct{})
	maxCoord := 1 << r.zoom
	for x := int(math.Floor(r.x0 / 256)); x <= int(math.Floor(r.x0+r.width))/256; x++ {
		for y := int(math.Floor(r.y0 / 256)); y <= int(math.Floor(r.y0+r.height))/256; y++ {
			if y >= 0 && y < maxCoord {
				tileCoords[osm.TileCoord{X: x % maxCoord, Y: y, Zoom: r.zoom}] = struct{}{}
			}
		}
	}
	r.mutex.Lock()
	for coords := range r.visibleTiles {
		if _, ok := tileCoords[coords]; !ok {
			delete(r.visibleTiles, coords)
		} else {
			delete(tileCoords, coords)
		}
	}
	for coords := range r.visiblePlaceholderTiles {
		if _, ok := tileCoords[coords]; !ok {
			delete(r.visiblePlaceholderTiles, coords)
		}
	}
	// in tileCoords leave only new tiles to fetch.
	for coords := range r.tilesToFetch {
		if _, ok := tileCoords[coords]; !ok {
			delete(r.tilesToFetch, coords)
		} else {
			delete(tileCoords, coords)
		}
	}
	for coords := range tileCoords {
		r.tilesToFetch[coords] = struct{}{}
	}
	r.mutex.Unlock()
	for coords := range tileCoords {
		r.fetchIfNotCached(coords)
	}
}

func (r *SolutionMap) fetchTile(coords osm.TileCoord) {
	img, err := r.tileFetcher.GetTile(coords)
	if err != nil {
		if errors.Is(err, osm.ErrBusy) {
			r.onFetchBusy(coords)
		} else {
			fmt.Println("Other error: ", err)
		}
		return
	}

	r.onTileRead(coords, img)
}
func (r *SolutionMap) fetchIfNotCached(coords osm.TileCoord) {
	tileImage, hasTile := r.getFromCache(coords)
	placeHolder := false
	if !hasTile {
		if coords.Zoom > 0 {
			zoomedOutCoords := osm.TileCoord{X: coords.X / 2, Y: coords.Y / 2, Zoom: coords.Zoom - 1}
			if zoomedOutTileImage, ok := r.getFromCache(zoomedOutCoords); ok {
				sourceX := (coords.X % 2) * 128
				sourceY := (coords.Y % 2) * 128
				zoomedImage := image.NewNRGBA(image.Rect(0, 0, 256, 256))
				draw.NearestNeighbor.Scale(zoomedImage, zoomedImage.Bounds(),
					zoomedOutTileImage, image.Rect(sourceX, sourceY, sourceX+128, sourceY+128),
					draw.Over, nil)
				tileImage = zoomedImage
				placeHolder = true
			}
		}
	}
	if tileImage != nil {
		r.addTile(coords, tileImage, placeHolder)
	}
	if !hasTile {
		go r.fetchTile(coords)
	} else {
		r.removeMissingTile(coords)
	}
}

func (r *solutionMapRenderer) geoToScreenCoordinates(x, y float32) (float32, float32) {
	zoomPow := math.Pow(2., float32(r.solutionMap.zoom))
	return x*zoomPow*256.0 - r.solutionMap.x0, y*zoomPow*256.0 - r.solutionMap.y0
}

func (s *SolutionMap) ScrollToPortal(guid string) {
	// 	portal, ok := s.portals[guid]
	// 	if !ok {
	// 		log.Println("Cannot locate portal", guid, "on the map")
	// 		return
	// 	}
	// 	x, y := s.GeoToScreenCoordinates(portal.coords.X, portal.coords.Y)
	// 	if x >= 0 && x < float64(s.canvas.Width()) &&
	// 		y >= 0 && y < float64(s.canvas.Height()) {
	// 		return
	// 	}
	// 	s.shiftMap(x-float64(s.canvas.Width())/2, y-float64(s.canvas.Height())/2)
}

// func (s *SolutionMap) OnPortalLeftClick(onPortalLeftClick func(string)) {
// 	s.onPortalLeftClick = onPortalLeftClick
// }
// func (s *SolutionMap) OnPortalRightClick(onPortalRightClick func(string, int, int)) {
// 	s.onPortalRightClick = onPortalRightClick
// }
func (s *SolutionMap) SetPortals(portals []lib.Portal) {
	if len(portals) == 0 {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.portals = nil
		s.portalIndices = make(map[string]int)
		return
	}
	chQuery := s2.NewConvexHullQuery()
	for _, portal := range portals {
		chQuery.AddPoint(s2.PointFromLatLng(portal.LatLng))
	}
	chQuery.ConvexHull()

	minX, minY, maxX, maxY := float32(math.MaxFloat32), float32(math.MaxFloat32), float32(-math.MaxFloat32), float32(-math.MaxFloat32)
	for _, portal := range portals {
		mapCoords := projection.FromLatLng(portal.LatLng)
		mapCoords.X = (mapCoords.X + 180) / 360
		mapCoords.Y = (180 - mapCoords.Y) / 360

		minX = math.Min(float32(mapCoords.X), minX)
		minY = math.Min(float32(mapCoords.Y), minY)
		maxX = math.Max(float32(mapCoords.X), maxX)
		maxY = math.Max(float32(mapCoords.Y), maxY)
	}
	numTilesX := math.Ceil(s.width / 256.)
	numTilesY := math.Ceil(s.height / 256.)
	for s.zoom = 19; s.zoom >= 0; s.zoom-- {
		zoomPow := math.Pow(2., float32(s.zoom))
		minXTile, minYTile := math.Floor(minX*zoomPow), math.Floor(minY*zoomPow)
		maxXTile, maxYTile := math.Floor(maxX*zoomPow), math.Floor(maxY*zoomPow)
		if maxXTile-minXTile+1 <= numTilesX && maxYTile-minYTile+1 <= numTilesY {
			break
		}
	}
	if s.zoom < 0 {
		s.zoom = 0
	}
	zoomPow := math.Pow(2., float32(s.zoom))
	s.x0 = (maxX+minX)*zoomPow*0.5*256.0 - s.width*0.5
	s.y0 = (maxY+minY)*zoomPow*0.5*256.0 - s.height*0.5
	s.mutex.Lock()
	s.portals = []mapPortal{}
	s.portalIndices = make(map[string]int)
	s.portalShapeIndex = s2.NewShapeIndex()
	s.shapeIndexIdToPortalId = make(map[int32]int)
	for i, portal := range portals {
		mapCoords := projection.FromLatLng(portal.LatLng)
		mapCoords.X = (mapCoords.X + 180) / 360
		mapCoords.Y = (180 - mapCoords.Y) / 360

		//		x, y := s.GeoToScreenCoordinates(float32(mapCoords.X), float32(mapCoords.Y))
		//		item := canvas.NewCircle(color.NRGBA{255, 170, 0, 128})
		//		item.Resize(fyne.Size{13, 13})
		//		item.StrokeColor = color.Black
		//		item.Move(fyne.Position{x - 7, y - 7}) //s.canvas.CreateOval(x-4, y-4, x+5, y+5, tk.CanvasItemAttrFill("orange"), tk.CanvasItemAttrTags([]string{"portal"}))
		// 	item.Raise()
		guid := portal.Guid // local copy to make closure captures work correctly
		s.portals = append(s.portals, mapPortal{GUID: guid, Coords: mapCoords, Name: portal.Name, Color: color.NRGBA{255, 170, 0, 128}, StrokeColor: color.NRGBA{255, 170, 0, 255}})
		s.portalIndices[guid] = len(s.portals) - 1
		portalPoint := s2.PointFromLatLng(portal.LatLng)
		portalCells := s2.SimpleRegionCovering(portalPoint, portalPoint, 30)
		cell := s2.CellFromCellID(portalCells[0])
		portalId := s.portalShapeIndex.Add(s2.PolygonFromCell(cell))
		s.shapeIndexIdToPortalId[portalId] = i
		// 	item.BindEvent("<Button-1>", func(e *tk.Event) {
		// 		if s.onPortalLeftClick != nil {
		// 			s.onPortalLeftClick(guid)
		// 		}
		// 	})
		// 	item.BindEvent("<Button-3>", func(e *tk.Event) {
		// 		if s.onPortalRightClick != nil {
		// 			s.onPortalRightClick(guid, e.GlobalPosX, e.GlobalPosY)
		// 		}
		// 	})
		// 	item.BindEvent("<Enter>", func(e *tk.Event) {
		// 		s.onPortalEntered(guid)
		// 	})
		// 	item.BindEvent("<Leave>", func(e *tk.Event) {
		// 		s.onPortalLeft(guid)
		// 	})
	}
	s.mutex.Unlock()
	s.updateVisibleTiles()
	s.Refresh()
}

func (s *SolutionMap) SetSolution(lines [][]lib.Portal) {
	points := make([][]s2.Point, 0, len(lines))
	for _, line := range lines {
		linePoints := make([]s2.Point, 0, len(line))
		for _, portal := range line {
			linePoints = append(linePoints, s2.PointFromLatLng(portal.LatLng))
		}
		points = append(points, linePoints)
	}
	s.setSolutionPoints(points)
	s.Refresh()
}

func (s *SolutionMap) setSolutionPoints(lines [][]s2.Point) {
	tesselator := s2.NewEdgeTessellator(projection, 1e-3)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	portalPaths := make([][]r2.Point, 0, len(lines))
	for _, line := range lines {
		path := []r2.Point{}
		for i := 1; i < len(line); i++ {
			path = tesselator.AppendProjected(line[i-1], line[i], path)
		}
		for i := range path {
			path[i].X = (path[i].X + 180) / 360
			path[i].Y = (180 - path[i].Y) / 360
		}
		portalPaths = append(portalPaths, path)
	}
	s.portalPaths = &portalPaths
	// 	s.showSolution()
	// 	if len(s.lines) > 0 {
	//		s.canvas.RaiseItemsAbove("portal", "link")
	//	}
}
