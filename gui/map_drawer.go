package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"sync"
	"time"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/golang/geo/r2"
	"github.com/golang/geo/s2"
	"github.com/golang/groupcache/lru"
	"github.com/inkyblackness/imgui-go/v3"
	guigl "github.com/pwiecz/portal_patterns/gui/gl"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
	"golang.org/x/image/draw"
)

const PortalCircleRadius = 7

var projection = s2.NewMercatorProjection(180)

type showTileRequest struct {
	coord osm.TileCoord
	tile  image.Image
}

type mapPortal struct {
	coords      r2.Point
	fillColor   imgui.PackedColor
	strokeColor imgui.PackedColor
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

type glTexture uint32
type MapDrawer struct {
	imguiContext         *imgui.Context
	imguiRenderer        *guigl.OpenGL2
	initialized          bool
	tileCache            *lockedTileCache
	mapTiles             map[osm.TileCoord]glTexture
	missingTiles         *lockedCoordSet
	portals              []mapPortal
	portalShapeIndex     *s2.ShapeIndex
	shapeIndexIDToPortal map[int32]int
	paths                [][]r2.Point
	portalIndices        map[string]int
	portalDrawOrder      []int
	defaultPortalColor   imgui.PackedColor
	//	showTileChannel       chan showTileRequest
	//	setPortalsChannel     chan []lib.Portal
	//	setPathsChannel       chan [][]s2.Point
	taskQueue             TaskQueue
	asyncChannel          chan struct{}
	tileFetcher           *osm.MapTiles
	width, height         float32
	zoom                  int
	zoomPow               float64
	x0, y0                float64
	mouseX, mouseY        int
	portalUnderMouse      int
	onMapChangedCallbacks []func()
}

func NewMapDrawer(width, height int, tileFetcher *osm.MapTiles) *MapDrawer {
	w := &MapDrawer{
		tileCache:          newLockedTileCache(1000),
		mapTiles:           make(map[osm.TileCoord]glTexture),
		missingTiles:       newLockedCoordSet(),
		defaultPortalColor: imgui.Packed(color.NRGBA{255, 127, 0, 127}),
		tileFetcher:        tileFetcher,
		portalUnderMouse:   -1,
		portalIndices:      make(map[string]int),
		width:              float32(width),
		height:             float32(height),
	}
	return w
}

func (w *MapDrawer) Async(callback func()) {
	w.taskQueue.Enqueue(callback)
	for _, callback := range w.onMapChangedCallbacks {
		callback()
	}
}
func (w *MapDrawer) SetPortalColor(guid string, fillColor, strokeColor color.Color) {
	w.Async(func() {
		w.portals[w.portalIndices[guid]].fillColor = imgui.Packed(fillColor)
		w.portals[w.portalIndices[guid]].strokeColor = imgui.Packed(strokeColor)
		go w.MapChanged()
	})
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
}
func (w *MapDrawer) Resize(width, height int) {
	w.Async(func() {
		w.width = float32(width)
		w.height = float32(height)
		w.redrawTiles()
	})
}
func (w *MapDrawer) Drag(dx, dy int) {
	w.Async(func() {
		w.x0 += float64(dx)
		w.y0 += float64(dy)
		w.redrawTiles()
	})
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
}
func (w *MapDrawer) Hover(x, y int) {
	w.Async(func() {
		w.mouseX, w.mouseY = x, y
		if len(w.portals) == 0 {
			return
		}
		mapX := (float64(x) + w.x0) / 256 / w.zoomPow
		mapY := (float64(y) + w.y0) / 256 / w.zoomPow
		projectedX := mapX*360 - 180
		projectedY := 180 - mapY*360
		point := projection.Unproject(r2.Point{projectedX, projectedY})
		opts := s2.NewClosestEdgeQueryOptions().MaxResults(1)
		query := s2.NewClosestEdgeQuery(w.portalShapeIndex, opts)
		target := s2.NewMinDistanceToPointTarget(point)
		result := query.FindEdges(target)
		if len(result) == 0 {
			return
		}
		shapeID := result[0].ShapeID()
		portalIx, ok := w.shapeIndexIDToPortal[shapeID]
		if !ok {
			return
		}
		closestPortal := w.portals[portalIx]
		dx, dy := mapX-closestPortal.coords.X, mapY-closestPortal.coords.Y
		dx, dy = dx*256*w.zoomPow, dy*256*w.zoomPow
		portalUnderMouse := -1
		if dx*dx+dy*dy <= PortalCircleRadius*PortalCircleRadius {
			portalUnderMouse = portalIx
		}
		if portalUnderMouse != w.portalUnderMouse {
			w.portalUnderMouse = portalUnderMouse
			w.MapChanged()
		}
	})
}
func (w *MapDrawer) Leave() {
	w.Async(func() {
		w.portalUnderMouse = -1
	})
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
	context := imgui.CreateContext(nil)
	imgui.CurrentIO().SetDisplaySize(imgui.Vec2{float32(screenWidth), float32(screenHeight)})
	renderer, err := guigl.NewOpenGL2(imgui.CurrentIO())
	if err != nil {
		panic(err)
	}

	w.imguiContext = context
	w.imguiRenderer = renderer
	//	w.InitOpenGL()
	w.onNewPortals(nil)
	w.initialized = true
}
func (w *MapDrawer) Update() {
	for !w.taskQueue.Empty() {
		callback := w.taskQueue.Dequeue()
		callback()
	}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	if err := gl.GetError(); err != gl.NO_ERROR {
		log.Fatalln("error on start", err)
	}
	w.drawAllTilesImgui()
	w.drawAllPortalsImgui()
	w.drawAllPathsImgui()
	w.drawPortalLabelImgui()
}

func (w *MapDrawer) onNewPortals(portals []lib.Portal) {
	minX, minY, maxX, maxY := math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for _, portal := range portals {
		mapCoords := projection.FromLatLng(portal.LatLng)
		mapCoords.X = (mapCoords.X + 180) / 360
		mapCoords.Y = (180 - mapCoords.Y) / 360
		minX = math.Min(mapCoords.X, minX)
		minY = math.Min(mapCoords.Y, minY)
		maxX = math.Max(mapCoords.X, maxX)
		maxY = math.Max(mapCoords.Y, maxY)
	}
	numTilesX := math.Ceil(float64(w.width) / 256.)
	numTilesY := math.Ceil(float64(w.height) / 256.)
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
	w.x0 = (maxX+minX)*w.zoomPow*0.5*256.0 - float64(w.width)*0.5
	w.y0 = (maxY+minY)*w.zoomPow*0.5*256.0 - float64(w.height)*0.5
	w.portals = make([]mapPortal, 0, len(portals))
	w.portalShapeIndex = s2.NewShapeIndex()
	w.shapeIndexIDToPortal = make(map[int32]int)
	w.portalDrawOrder = w.portalDrawOrder[:0]
	if len(portals) == 0 {
		w.zoom = 0
		w.zoomPow = 1
		w.x0 = 0
		w.y0 = 0
		w.portalUnderMouse = -1
		w.redrawTiles()
		w.MapChanged()
		return
	}
	for i, portal := range portals {
		portalPoint := s2.PointFromLatLng(portal.LatLng)
		portalCells := s2.SimpleRegionCovering(portalPoint, portalPoint, 30)
		if len(portalCells) != 1 {
			panic(portalCells)
		}
		cell := s2.CellFromCellID(portalCells[0])
		portalID := w.portalShapeIndex.Add(s2.PolygonFromCell(cell))
		w.shapeIndexIDToPortal[portalID] = i
		mapCoords := projection.FromLatLng(portal.LatLng)
		mapCoords.X = (mapCoords.X + 180) / 360
		mapCoords.Y = (180 - mapCoords.Y) / 360
		w.portals = append(w.portals, mapPortal{
			coords:      mapCoords,
			fillColor:   w.defaultPortalColor,
			strokeColor: imgui.Packed(color.NRGBA{0, 0, 0, 0}),
			name:        portal.Name,
			guid:        portal.Guid,
			drawOrder:   i,
		})
		w.portalIndices[portal.Guid] = i
		w.portalDrawOrder = append(w.portalDrawOrder, i)
	}
	w.portalUnderMouse = -1

	w.redrawTiles()
	w.MapChanged()
}

const LabelXMargin = 10

func (w *MapDrawer) drawPortalLabelImgui() {
	if w.portalUnderMouse < 0 || w.portalUnderMouse >= len(w.portals) {
		return
	}
	imgui.NewFrame()
	drawList := imgui.BackgroundDrawList()
	portal := w.portals[w.portalUnderMouse]
	x := float32(portal.coords.X*w.zoomPow*256 - w.x0)
	y := float32(portal.coords.Y*w.zoomPow*256 - w.y0)
	textSize := imgui.CalcTextSize(portal.name, false, 0)
	labelPosX, labelPosY := x-textSize.X/2-LabelXMargin, y-PortalCircleRadius-2
	if labelPosX < 0 {
		labelPosX = 0
	} else if labelPosX+textSize.X >= w.width {
		labelPosX = w.width - textSize.X - LabelXMargin*2
	}
	if labelPosY-textSize.Y < 0 {
		labelPosY = y + 5 + textSize.Y + 4
	}
	labelPos := imgui.Vec2{labelPosX, labelPosY - 20}
	white := imgui.Packed(color.NRGBA{255, 255, 255, 255})
	drawList.AddRectFilled(labelPos, labelPos.Plus(imgui.Vec2{textSize.X + 2*LabelXMargin, textSize.Y}), white)
	black := imgui.Packed(color.NRGBA{0, 0, 0, 255})
	textPos := labelPos.Plus(imgui.Vec2{LabelXMargin, 0})
	drawList.AddText(textPos, black, portal.name)
	imgui.Render()
	size := [2]float32{w.width, w.height}
	w.imguiRenderer.Render(size, size, imgui.RenderedDrawData())
}
func (w *MapDrawer) drawAllTilesImgui() {
	imgui.NewFrame()
	drawList := imgui.BackgroundDrawList()
	for coord, tex := range w.mapTiles {
		dx := float32(coord.X)*256 - float32(w.x0)
		dy := float32(coord.Y)*256 - float32(w.y0)
		drawList.AddImage(imgui.TextureID(tex), imgui.Vec2{dx, dy}, imgui.Vec2{dx + 256, dy + 256})
	}
	imgui.Render()
	size := [2]float32{w.width, w.height}
	w.imguiRenderer.Render(size, size, imgui.RenderedDrawData())
}
func (w *MapDrawer) drawAllPortalsImgui() {
	imgui.NewFrame()
	drawList := imgui.BackgroundDrawList()
	size := [2]float32{w.width, w.height}
	for i, portalIndex := range w.portalDrawOrder {
		portal := w.portals[portalIndex]
		x := float32(portal.coords.X*w.zoomPow*256 - w.x0)
		y := float32(portal.coords.Y*w.zoomPow*256 - w.y0)
		drawList.AddCircleFilled(imgui.Vec2{x, y}, PortalCircleRadius, portal.fillColor)
		drawList.AddCircle(imgui.Vec2{x, y}, PortalCircleRadius, portal.strokeColor)
		// Split drawing portals into smaller chunks, otherwise we exceed imgui limits.
		if i%499 == 1 {
			imgui.Render()
			w.imguiRenderer.Render(size, size, imgui.RenderedDrawData())
			imgui.NewFrame()
		}
	}
	imgui.Render()
	w.imguiRenderer.Render(size, size, imgui.RenderedDrawData())
}
func (w *MapDrawer) drawAllPathsImgui() {
	imgui.NewFrame()
	purple := imgui.Packed(color.NRGBA{100, 50, 225, 175})
	drawList := imgui.BackgroundDrawList()
	for _, path := range w.paths {
		for i := 1; i < len(path); i++ {
			x0 := float32(path[i-1].X*w.zoomPow*256 - w.x0)
			y0 := float32(path[i-1].Y*w.zoomPow*256 - w.y0)
			x1 := float32(path[i].X*w.zoomPow*256 - w.x0)
			y1 := float32(path[i].Y*w.zoomPow*256 - w.y0)
			drawList.AddLineV(imgui.Vec2{x0, y0}, imgui.Vec2{x1, y1}, purple, 3)
		}
	}
	imgui.Render()
	size := [2]float32{w.width, w.height}
	w.imguiRenderer.Render(size, size, imgui.RenderedDrawData())
}
func (w *MapDrawer) SetPortals(portals []lib.Portal) {
	w.Async(func() { w.onNewPortals(portals) })
	//	go func() {
	//		for _, callback := range w.onMapChangedCallbacks {
	//			callback()
	//		}
	//		w.setPortalsChannel <- portals
	//	}()
}
func (w *MapDrawer) SetPaths(paths [][]s2.Point) {
	w.Async(func() {
		tesselator := s2.NewEdgeTessellator(projection, 1e-3)
		fmt.Println("SetPaths", paths, w.paths)
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
		w.MapChanged()
	})
	//	go func() {
	//		for _, callback := range w.onMapChangedCallbacks {
	//			callback()
	//		}
	//		w.setPathsChannel <- paths
	//	}()
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
	//	go func() {
	w.Async(func() {
		//		for _, callback := range w.onMapChangedCallbacks {
		//			callback()
		//		}
		//		w.showTileChannel <- showTileRequest{coord: coord, tile: img}
		w.showTile(coord, img)
	})
	//	}()
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
				tileCoords[osm.TileCoord{x, y, w.zoom}] = struct{}{}
			}
		}
	}
	for coord, tex := range w.mapTiles {
		if _, ok := tileCoords[coord]; !ok {
			deleteTexture(tex)
			delete(w.mapTiles, coord)
		} else {
			delete(tileCoords, coord)
		}
	}
	w.missingTiles.Clear()
	for coord := range tileCoords {
		w.tryShowTile(coord)
	}
}
func (w *MapDrawer) showTile(coord osm.TileCoord, img image.Image) {
	if tex, ok := w.mapTiles[coord]; ok {
		deleteTexture(tex)
	}
	w.mapTiles[coord] = newTexture(img)
	for _, callback := range w.onMapChangedCallbacks {
		callback()
	}
}

func (w *MapDrawer) fetchTile(coord osm.TileCoord) {
	for {
		img, err := w.tileFetcher.GetTile(coord)
		if err == nil {
			w.onTileRead(coord, img)
			return
		}
		if !errors.Is(err, osm.ErrBusy) {
			if !errors.Is(err, osm.ErrAlreadyRequested) {
				fmt.Println("fetching error:", err)
			}
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
	if tileImage == nil {
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

func deleteTexture(tex glTexture) {
	texIx := uint32(tex)
	gl.DeleteTextures(1, &texIx)
}
func newTexture(img image.Image) glTexture {
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(img.Bounds())
		if rgba.Stride != rgba.Rect.Size().X*4 {
			panic("unsupported stride")
		}
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.Enable(gl.TEXTURE_2D)
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return glTexture(texture)
}
