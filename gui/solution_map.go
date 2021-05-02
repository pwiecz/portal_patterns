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

var projection = s2.NewMercatorProjection(180)

type tile struct {
	x, y, zoom int
	image      image.Image
}

type mapPortal struct {
	Coords      r2.Point
	Name        string
	Color       color.Color
	StrokeColor color.Color
	//	Shape  *canvas.Circle
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

	portalPaths *[][]r2.Point
	tileFetcher *osm.MapTiles
	mutex       sync.Mutex
	tileCache   *lru.Cache
	//	lines              []*tk.CanvasLine
	visibleTiles            map[osm.TileCoord]image.Image
	visiblePlaceholderTiles map[osm.TileCoord]image.Image
	tilesToFetch            map[osm.TileCoord]struct{}

	portals map[string]mapPortal
	//	objects            []fyne.CanvasObject // cache for list of displayed canvas objects
	//	onPortalLeftClick  func(string)
	//	onPortalRightClick func(string, int, int)
}

func NewSolutionMap(tileFetcher *osm.MapTiles) *SolutionMap {
	s := &SolutionMap{
		tileFetcher:             tileFetcher,
		tileCache:               lru.New(1000),
		visibleTiles:            make(map[osm.TileCoord]image.Image),
		visiblePlaceholderTiles: make(map[osm.TileCoord]image.Image),
		tilesToFetch:            make(map[osm.TileCoord]struct{}),
		portals:                 make(map[string]mapPortal),
		portalPaths:             &[][]r2.Point{},
	}
	s.ExtendBaseWidget(s)
	return s
	//	s.Window = fyne.CurrentApp().NewWindow(name + " - © OpenStreetMap")
	//	s.layout = tk.NewVPackLayout(s.Window)
	//	s.canvas = tk.NewCanvas(s.Window, tk.CanvasAttrBackground("#C8C8C8"))
	//	s.layout.AddWidgetEx(s.canvas, tk.FillBoth, true, tk.AnchorNorth)
	//	s.Window.ResizeN(800, 600)
	//	s.canvas.BindEvent("<Configure>", func(e *tk.Event) {
	//		s.canvas.SetWidth(e.Width)
	//		s.canvas.SetHeight(e.Height)
	//		s.showTiles()
	//	})
	//	s.canvas.SetFocus()
	//	s.canvas.BindEvent("<Button1-Motion>", func(e *tk.Event) { s.OnDrag(e) })
	//s.canvas.BindEvent("<Control-Button-1>", func(e *tk.Event) { fmt.Println("Ctrl-Click") })
	//	s.canvas.BindEvent("<ButtonPress-1>", func(e *tk.Event) { s.OnButtonPress(e) })
	//	s.canvas.BindEvent("<ButtonPress-4>", func(e *tk.Event) { s.OnScrollUp(e) })
	//	s.canvas.BindEvent("<ButtonPress-5>", func(e *tk.Event) { s.OnScrollDown(e) })
	//	s.canvas.BindEvent("<MouseWheel>", func(e *tk.Event) {
	//		if e.WheelDelta < 0 {
	//			s.OnScrollDown(e)
	//		} else {
	//			s.OnScrollUp(e)
	//		}
	//	})
	//	s.canvas.BindEvent("<KeyRelease-plus>", func(e *tk.Event) {
	//		s.OnZoomIn(s.canvas.Width()/2, s.canvas.Height()/2)
	//	})
	//	s.canvas.BindEvent("<KeyRelease-minus>", func(e *tk.Event) {
	//		s.OnZoomOut(s.canvas.Width()/2, s.canvas.Height()/2)
	//	})
	//	s.portals = make(map[string]mapPortal)
	//	s.mapTiles = make(map[osm.TileCoord]*tk.CanvasImage)
	//	s.missingTiles = make(map[osm.TileCoord]bool)
	//	s.tileFetcher = tileFetcher
	//	s.tileCache = lru.New(1000)
	//	s.font = tk.LoadSysFont(tk.SysTextFont)
	//	s.fontDescription = s.font.Description()
	//	s.textAscent = s.font.Ascent()
	//	s.textDescent = s.font.Descent()
	//	s.layout.Repack()
}

type solutionMapRenderer struct {
	background   *canvas.Rectangle
	visibleTiles map[osm.TileCoord]*canvas.Image
	circles      map[string]*canvas.Circle
	paths        *[][]r2.Point // paths for which the lines were created
	linesZoom    int           // zoom level at which lines were drawn
	lines        []*canvas.Line
	solutionMap  *SolutionMap
}

func (m *SolutionMap) CreateRenderer() fyne.WidgetRenderer {
	return &solutionMapRenderer{
		solutionMap:  m,
		visibleTiles: make(map[osm.TileCoord]*canvas.Image),
		circles:      make(map[string]*canvas.Circle),
		background:   canvas.NewRectangle(color.White),
	}
}

func (r *solutionMapRenderer) Layout(size fyne.Size) {
	r.solutionMap.width = size.Width
	r.solutionMap.height = size.Height
	r.background.Resize(size)
	r.solutionMap.updateVisibleTiles()
}
func (r *solutionMapRenderer) MinSize() fyne.Size {
	return fyne.Size{256, 256}
}
func (r *solutionMapRenderer) Refresh() {
	r.solutionMap.mutex.Lock()
	for coords, _ := range r.visibleTiles {
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
	for guid, _ := range r.circles {
		if _, ok := r.solutionMap.portals[guid]; !ok {
			delete(r.circles, guid)
		}
	}
	for guid, portal := range r.solutionMap.portals {
		x, y := r.geoToScreenCoordinates(float32(portal.Coords.X), float32(portal.Coords.Y))
		if circle, ok := r.circles[guid]; ok {
			circle.FillColor = portal.Color
			circle.StrokeColor = portal.StrokeColor
			circle.Move(fyne.Position{x - 7, y - 7})
		} else {
			circle := canvas.NewCircle(portal.Color)
			circle.StrokeColor = portal.StrokeColor
			circle.StrokeWidth = 2
			circle.Resize(fyne.Size{13, 13})
			circle.Move(fyne.Position{x - 7, y - 7})
			r.circles[guid] = circle
		}
	}
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
				line.Position1 = fyne.Position{x0, y0}
				line.Position2 = fyne.Position{x1, y1}
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
				line.Position1 = fyne.Position{x0, y0}
				line.Position2 = fyne.Position{x1, y1}
				if r.linesZoom != r.solutionMap.zoom {
					line.Resize(fyne.Size{x1 - x0, y1 - y0})
				}
				lineIx++
			}
		}
		r.linesZoom = r.solutionMap.zoom
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
	img.Move(fyne.Position{x, y})
}
func (r *solutionMapRenderer) createImageFromImage(img image.Image) *canvas.Image {
	cImg := canvas.NewImageFromImage(img)
	cImg.FillMode = canvas.ImageFillOriginal
	cImg.ScaleMode = canvas.ImageScaleFastest
	cImg.Resize(fyne.Size{256, 256})
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
	return objects
}
func (r *solutionMapRenderer) Destroy() {}
func (r *SolutionMap) isTileMissing(coords osm.TileCoord) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.tilesToFetch[coords]
	return ok
}
func (r *SolutionMap) addMissingTile(coords osm.TileCoord) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.tilesToFetch[coords] = struct{}{}
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

func (s *SolutionMap) SetPortalColor(guid string, color color.Color) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if portal, ok := s.portals[guid]; ok {
		portal.Color = color
	}
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
	s.portals = make(map[string]mapPortal)
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
}
func (s *SolutionMap) MouseOut() {}
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
	for coords, _ := range r.visibleTiles {
		if _, ok := tileCoords[coords]; !ok {
			delete(r.visibleTiles, coords)
		} else {
			delete(tileCoords, coords)
		}
	}
	for coords, _ := range r.visiblePlaceholderTiles {
		if _, ok := tileCoords[coords]; !ok {
			delete(r.visiblePlaceholderTiles, coords)
		}
	}
	// in tileCoords leave only new tiles to fetch.
	for coords, _ := range r.tilesToFetch {
		if _, ok := tileCoords[coords]; !ok {
			delete(r.tilesToFetch, coords)
		} else {
			delete(tileCoords, coords)
		}
	}
	for coords, _ := range tileCoords {
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

// func (s *SolutionMap) showSolution() {
// 	for _, line := range s.lines {
// 		s.canvas.DeleteLine(line)
// 	}
// 	s.lines = []*tk.CanvasLine{}
// 	for _, path := range s.portalPaths {
// 		coords := make([]tk.CanvasCoordinates, 0, len(path))
// 		for _, point := range path {
// 			x, y := s.GeoToScreenCoordinates(point.X, point.Y)
// 			coords = append(coords, tk.CanvasPixelCoords(x, y))
// 		}
// 		line := s.canvas.CreateLine(coords, tk.CanvasItemAttrWidth(1), tk.CanvasItemAttrFill("blue"), tk.CanvasItemAttrTags([]string{"link"}))
// 		s.lines = append(s.lines, line)
// 	}

// }
func (s *SolutionMap) setItemCoords() {
	// for _, portal := range s.portals {
	// 	x, y := s.GeoToScreenCoordinates(portal.Coords.X, portal.Coords.Y)
	// 	err := portal.shape.MoveTo(x-4, y-4)
	// 	if err != nil {
	// 		log.Println(err)
	// 	}
	// }
}

func (r *solutionMapRenderer) geoToScreenCoordinates(x, y float32) (float32, float32) {
	zoomPow := math.Pow(2., float32(r.solutionMap.zoom))
	return x*zoomPow*256.0 - r.solutionMap.x0, y*zoomPow*256.0 - r.solutionMap.y0
}

// func (s *SolutionMap) ScrollToPortal(guid string) {
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
// }
// func (s *SolutionMap) OnScrollUp(e *tk.Event) {
// 	s.OnZoomIn(e.PosX, e.PosY)
// }
// func (s *SolutionMap) OnScrollDown(e *tk.Event) {
// 	s.OnZoomOut(e.PosX, e.PosY)
// }
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
		s.portals = make(map[string]mapPortal)
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
	s.portals = make(map[string]mapPortal)
	for _, portal := range portals {
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
		s.portals[guid] = mapPortal{Coords: mapCoords, Name: portal.Name, Color: color.NRGBA{255, 170, 0, 128}, StrokeColor: color.NRGBA{255, 170, 0, 255}}
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

// func (s *SolutionMap) onPortalEntered(guid string) {
// 	if s.nameLabel != nil {
// 		s.canvas.DeleteText(s.nameLabel)
// 		s.nameLabel = nil
// 	}
// 	if s.nameLabelBackgound != nil {
// 		s.canvas.DeleteRectangle(s.nameLabelBackgound)
// 		s.nameLabelBackgound = nil
// 	}
// 	portal, ok := s.portals[guid]
// 	if !ok {
// 		return
// 	}
// 	x, y := s.GeoToScreenCoordinates(portal.coords.X, portal.coords.Y)
// 	backgroundWidth := float64(s.font.MeasureTextWidth(portal.name) + 6)
// 	backgroundHeight := float64(s.textAscent + s.textDescent + 8)
// 	backgroundX, backgroundY := x-backgroundWidth/2, y-9
// 	if backgroundX < 0 {
// 		backgroundX = 0
// 	} else if backgroundX+backgroundWidth >= float64(s.canvas.Width()) {
// 		backgroundX = float64(s.canvas.Width()) - backgroundWidth
// 	}
// 	if backgroundY-backgroundHeight < 0 {
// 		backgroundY = y + 5 + backgroundHeight + 4
// 	}
// 	s.nameLabelBackgound = s.canvas.CreateRectangle(backgroundX, backgroundY, backgroundX+backgroundWidth, backgroundY-backgroundHeight, tk.CanvasItemAttrFill("white"))
// 	s.nameLabel = s.canvas.CreateText(backgroundX+backgroundWidth/2, backgroundY-3, tk.CanvasItemAttrText(portal.name), tk.CanvasItemAttrFont(s.fontDescription), tk.CanvasItemAttrAnchor(tk.AnchorSouth))
// }
// func (s *SolutionMap) onPortalLeft(guid string) {
// 	if s.nameLabel != nil {
// 		s.canvas.DeleteText(s.nameLabel)
// 		s.nameLabel = nil
// 	}
// 	if s.nameLabelBackgound != nil {
// 		s.canvas.DeleteRectangle(s.nameLabelBackgound)
// 		s.nameLabelBackgound = nil
// 	}
// }

func portalsToPoints(portals []lib.Portal) []s2.Point {
	points := make([]s2.Point, 0, len(portals))
	for _, portal := range portals {
		points = append(points, s2.PointFromLatLng(portal.LatLng))
	}
	return points
}
