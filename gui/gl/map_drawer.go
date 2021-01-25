package gl

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
	"github.com/pwiecz/imgui-go"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
	"golang.org/x/image/draw"
)

var projection = s2.NewMercatorProjection(180)

type showTileRequest struct {
	coord osm.TileCoord
	tile  image.Image
}

type mapPortal struct {
	coords r2.Point
	name   string
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

type MapDrawer struct {
	imguiContext          *imgui.Context
	imguiRenderer         *OpenGL2
	initialized           bool
	tileCache             *lockedTileCache
	mapTiles              map[osm.TileCoord]uint32
	missingTiles          *lockedCoordSet
	portals               []mapPortal
	portalIndex           s2.CellIndex
	paths                 [][]r2.Point
	selectedPortals       map[string]bool
	disabledPortals       map[string]bool
	showTileChannel       chan showTileRequest
	setPortalsChannel     chan []lib.Portal
	setPathsChannel       chan [][]lib.Portal
	tileFetcher           *osm.MapTiles
	zoom                  int
	zoomPow               float64
	x0, y0                float64
	onMapChangedCallbacks []func()
}

var initGLOnce sync.Once

func NewMapDrawer(tileFetcher *osm.MapTiles) *MapDrawer {
	context := imgui.CreateContext(nil)
	imgui.CurrentIO().SetDisplaySize(imgui.Vec2{800, 600})
	renderer, err := NewOpenGL2(imgui.CurrentIO())
	if err != nil {
		panic(err)
	}
	w := &MapDrawer{
		imguiContext:      context,
		imguiRenderer:     renderer,
		tileCache:         newLockedTileCache(1000),
		mapTiles:          make(map[osm.TileCoord]uint32),
		missingTiles:      newLockedCoordSet(),
		showTileChannel:   make(chan showTileRequest),
		setPortalsChannel: make(chan []lib.Portal),
		setPathsChannel:   make(chan [][]lib.Portal),
		tileFetcher:       tileFetcher,
	}
	return w
}

func (w *MapDrawer) Drag(dx, dy int) {
	w.x0 += float64(dx)
	w.y0 += float64(dy)
	w.redrawTiles()
}
func (w *MapDrawer) ZoomIn(x, y int) {
	if w.zoom < osm.MAX_ZOOM_LEVEL {
		w.zoom++
		w.zoomPow *= 2.0
		w.x0 = (w.x0+float64(x))*2.0 - float64(x)
		w.y0 = (w.y0+float64(y))*2.0 - float64(y)
		w.redrawTiles()
		fmt.Println("zoom in")
	}
}
func (w *MapDrawer) ZoomOut(x, y int) {
	if w.zoom > 0 {
		w.zoom--
		w.zoomPow /= 2.0
		w.x0 = (w.x0+float64(x))*0.5 - float64(x)
		w.y0 = (w.y0+float64(y))*0.5 - float64(y)
		w.redrawTiles()
		fmt.Println("zoom out")
	}
}
func (w *MapDrawer) OnMapChanged(callback func()) {
	w.onMapChangedCallbacks = append(w.onMapChangedCallbacks, callback)
}
func (w *MapDrawer) Init() {
	if w.initialized {
		return
	}
	//	w.InitOpenGL()
	w.initialized = true
}
func (w *MapDrawer) Update() {
	select {
	case req := <-w.showTileChannel:
		w.showTile(req.coord, req.tile)
	case portals := <-w.setPortalsChannel:
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
		numTilesX := math.Ceil(800. / 256.)
		numTilesY := math.Ceil(600. / 256.)
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
		w.x0 = (maxX+minX)*w.zoomPow*0.5*256.0 - float64(800)*0.5
		w.y0 = (maxY+minY)*w.zoomPow*0.5*256.0 - float64(600)*0.5
		w.portals = make([]mapPortal, 0, len(portals))
		w.portalIndex = s2.CellIndex{}
		for i, portal := range portals {
			portalPoint := s2.PointFromLatLng(portal.LatLng)
			portalCells := s2.SimpleRegionCovering(portalPoint, portalPoint, 30)
			if len(portalCells) != 1 {
				panic(portalCells)
			}
			w.portalIndex.Add(portalCells[0], int32(i))
			mapCoords := projection.FromLatLng(portal.LatLng)
			mapCoords.X = (mapCoords.X + 180) / 360
			mapCoords.Y = (180 - mapCoords.Y) / 360
			w.portals = append(w.portals, mapPortal{
				coords: mapCoords,
				name:   portal.Name,
			})
		}
		w.redrawTiles()
		for _, callback := range w.onMapChangedCallbacks {
			callback()
		}
	case paths := <-w.setPathsChannel:
		tesselator := s2.NewEdgeTessellator(projection, 1e-3)
		w.paths = w.paths[:0]
		for _, path := range paths {
			mapPath := []r2.Point{}
			for i := 1; i < len(path); i++ {
				mapPath = tesselator.AppendProjected(
					s2.PointFromLatLng(path[i-1].LatLng),
					s2.PointFromLatLng(path[i].LatLng),
					mapPath)
			}
			for i := range mapPath {
				mapPath[i].X = (mapPath[i].X + 180) / 360
				mapPath[i].Y = (180 - mapPath[i].Y) / 360
			}
			w.paths = append(w.paths, mapPath)
		}
		for _, callback := range w.onMapChangedCallbacks {
			callback()
		}
	default:
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		if err := gl.GetError(); err != gl.NO_ERROR {
			log.Fatalln("error on start", err)
		}
		w.DrawAllTilesImgui()
		w.DrawAllPortalsImgui()
		w.DrawAllPathsImgui()
	}
}

func (w *MapDrawer) DrawAllTilesImgui() {
	imgui.NewFrame()
	drawList := imgui.BackgroundDrawList()
	for coord, tex := range w.mapTiles {
		dx := float32(coord.X)*256 - float32(w.x0)
		dy := float32(coord.Y)*256 - float32(w.y0)
		drawList.AddImage(imgui.TextureID(tex), imgui.Vec2{dx, dy}, imgui.Vec2{dx + 256, dy + 256})
	}
	imgui.Render()
	//	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	//	clearColor := [3]float32{0.0, 0.0, 0.0}
	//	w.imguiRenderer.PreRender(clearColor)
	w.imguiRenderer.Render([2]float32{800, 600}, [2]float32{800, 600}, imgui.RenderedDrawData())
}
func (w *MapDrawer) DrawAllPortalsImgui() {
	imgui.NewFrame()
	orange := imgui.Packed(color.NRGBA{255, 127, 0, 255})
	black := imgui.Packed(color.NRGBA{0, 0, 0, 255})
	drawList := imgui.BackgroundDrawList()
	for i, portal := range w.portals {
		x := float32(portal.coords.X*w.zoomPow*256 - w.x0)
		y := float32(portal.coords.Y*w.zoomPow*256 - w.y0)
		drawList.AddCircleV(imgui.Vec2{x, y}, 5.1, black, 0, 0.1)
		drawList.AddCircleFilledV(imgui.Vec2{x, y}, 5, orange, 0)
		// Split drawing portals into smaller chunks, otherwise we exceed imgui limits.
		if i%499 == 1 {
			imgui.Render()
			w.imguiRenderer.Render([2]float32{800, 600}, [2]float32{800, 600}, imgui.RenderedDrawData())
			imgui.NewFrame()
		}
	}
	imgui.Render()
	w.imguiRenderer.Render([2]float32{800, 600}, [2]float32{800, 600}, imgui.RenderedDrawData())
}
func (w *MapDrawer) DrawAllPathsImgui() {
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
	w.imguiRenderer.Render([2]float32{800, 600}, [2]float32{800, 600}, imgui.RenderedDrawData())
}
func (w *MapDrawer) SetPortals(portals []lib.Portal) {
	go func() {
		for _, callback := range w.onMapChangedCallbacks {
			callback()
		}
		w.setPortalsChannel <- portals
	}()
}
func (w *MapDrawer) SetPaths(paths [][]lib.Portal) {
	go func() {
		for _, callback := range w.onMapChangedCallbacks {
			callback()
		}
		w.setPathsChannel <- paths
	}()
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
	go func() {
		for _, callback := range w.onMapChangedCallbacks {
			callback()
		}
		w.showTileChannel <- showTileRequest{coord: coord, tile: img}
	}()
}
func (w *MapDrawer) redrawTiles() {
	if w.zoomPow == 0 {
		return
	}
	tileCoords := make(map[osm.TileCoord]bool)
	maxCoord := 1 << w.zoom
	for x := int(math.Floor(w.x0 / 256)); x <= int(math.Floor(w.x0+800.))/256; x++ {
		for y := int(math.Floor(w.y0 / 256)); y <= int(math.Floor(w.y0+600.))/256; y++ {
			if y >= 0 && y < maxCoord {
				tileCoords[osm.TileCoord{x, y, w.zoom}] = true
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
	retry := 1
	for {
		img, err := w.tileFetcher.GetTile(coord)
		if err == nil {
			if retry > 1 {
				fmt.Println("retrying done")
			}
			w.onTileRead(coord, img)
			return
		}
		if !errors.Is(err, osm.ErrBusy) {
			fmt.Println("fetching error:", err)

			return
		}
		if !w.missingTiles.Contains(coord) {
			fmt.Println("Tile no longer needed 1")
			return
		}
		// try fetching again after 1 second
		timer := time.NewTimer(time.Second)
		fmt.Println("retrying", retry)
		retry++
		<-timer.C
		// check if we still need the tile before refetching
		if !w.missingTiles.Contains(coord) {
			fmt.Println("Tile no longer needed 2")
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

func deleteTexture(tex uint32) {
	gl.DeleteTextures(1, &tex)
}
func newTexture(img image.Image) uint32 {
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
	if err := gl.GetError(); err != gl.NO_ERROR {
		log.Fatalln("Error on GenTextures", err)
	}
	gl.BindTexture(gl.TEXTURE_2D, texture)
	if err := gl.GetError(); err != gl.NO_ERROR {
		log.Fatalln("Error on bind texture", err)
	}

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	if err := gl.GetError(); err != gl.NO_ERROR {
		log.Fatalln("Error on texparameteri", err)
	}

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
	if err := gl.GetError(); err != gl.NO_ERROR {
		log.Fatalln("Error on teximage2d", err)
	}

	return texture
}

var texCoords = []float32{
	0.0, 0.0, 0.0, 1.0, 1.0, 1.0, 1.0, 0.0,
}
var tileVertices = []float32{
	0.0, 0.0, 0.0,
	0.0, 256.0, 0.0,
	256.0, 256.0, 0.0,
	256.0, 0.0, 0.0,
}
