package gl

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/golang/geo/r2"
	"github.com/golang/geo/s2"
	"github.com/golang/groupcache/lru"
	"github.com/pwiecz/imgui-go"
	"github.com/pwiecz/portal_patterns/gui/osm"
	"github.com/pwiecz/portal_patterns/lib"
	"golang.org/x/image/draw"
)

var projection = s2.NewMercatorProjection(180) //lib.NewWebMercatorProjection()

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

type MapWindow struct {
	imguiContext           *imgui.Context
	imguiRenderer          *OpenGL2
	initialized            bool
	tileCache              *lockedTileCache
	mapTiles               map[osm.TileCoord]uint32
	missingTiles           *lockedCoordSet
	portals                []mapPortal
	paths                  [][]r2.Point
	selectedPortals        map[string]bool
	disabledPortals        map[string]bool
	showTileChannel        chan showTileRequest
	setPortalsChannel      chan []lib.Portal
	setPathsChannel        chan [][]lib.Portal
	tileFetcher            *osm.MapTiles
	zoom                   int
	zoomPow                float64
	x0, y0                 float64
	tileProgram            uint32
	tileProjectionLocation int32
	tileModelLocation      int32
	texLocation            int32
	tileVerticesVBO        uint32
	texPositionLocation    int32
	texCoordLocation       int32
	texVBO                 uint32
	meshProgram            uint32
	meshProjectionLocation int32
	meshModelLocation      int32
	meshPositionLocation   int32
	meshColorLocation      int32
	meshRadiusLocation     int32
	//	portalTex              uint32
	portalVerticesVBO     uint32
	projectionMatrix      mgl32.Mat4
	onMapChangedCallbacks []func()
}

var initGLOnce sync.Once

func NewMapWindow(title string, tileFetcher *osm.MapTiles) *MapWindow {
	context := imgui.CreateContext(nil)
	imgui.CurrentIO().SetDisplaySize(imgui.Vec2{800, 600})
	renderer, err := NewOpenGL2(imgui.CurrentIO())
	if err != nil {
		panic(err)
	}
	w := &MapWindow{
		imguiContext:  context,
		imguiRenderer: renderer,
		//		window:          window,
		tileCache:         newLockedTileCache(1000),
		mapTiles:          make(map[osm.TileCoord]uint32),
		missingTiles:      newLockedCoordSet(),
		showTileChannel:   make(chan showTileRequest),
		setPortalsChannel: make(chan []lib.Portal),
		setPathsChannel:   make(chan [][]lib.Portal),
		tileFetcher:       tileFetcher,
		projectionMatrix:  mgl32.Ident4(),
	}
	return w
}

func (w *MapWindow) Drag(dx, dy int) {
	w.x0 += float64(dx)
	w.y0 += float64(dy)
	w.redrawTiles()
}
func (w *MapWindow) ZoomIn(x, y int) {
	if w.zoom < osm.MAX_ZOOM_LEVEL {
		w.zoom++
		w.zoomPow *= 2.0
		w.x0 = (w.x0+float64(x))*2.0 - float64(x)
		w.y0 = (w.y0+float64(y))*2.0 - float64(y)
		w.redrawTiles()
		fmt.Println("zoom in")
	}
}
func (w *MapWindow) ZoomOut(x, y int) {
	if w.zoom > 0 {
		w.zoom--
		w.zoomPow /= 2.0
		w.x0 = (w.x0+float64(x))*0.5 - float64(x)
		w.y0 = (w.y0+float64(y))*0.5 - float64(y)
		w.redrawTiles()
		fmt.Println("zoom out")
	}
}
func (w *MapWindow) OnMapChanged(callback func()) {
	w.onMapChangedCallbacks = append(w.onMapChangedCallbacks, callback)
}
func (w *MapWindow) Init() {
	if w.initialized {
		return
	}
	//	w.InitOpenGL()
	w.initialized = true
}
func (w *MapWindow) Update() {
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
		for _, portal := range portals {
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
		w.paths = w.paths[:0]
		for _, path := range paths {
			mapPath := []r2.Point{}
			for _, portal := range path {
				mapCoords := projection.FromLatLng(portal.LatLng)
				mapCoords.X = (mapCoords.X + 180) / 360
				mapCoords.Y = (180 - mapCoords.Y) / 360
				mapPath = append(mapPath, mapCoords)
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
		//		w.DrawAllTiles()
		//		w.DrawAllPortals()
	}
}

func (w *MapWindow) InitOpenGL() {
	if err := gl.Init(); err != nil {
		log.Fatal("Cannot initialize OpenGL", err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr on glVersion", err)
	}
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.ClearColor(1.0, 1.0, 1.0, 1.0)
	var err error
	w.tileProgram, err = newProgram(tileVertexShader, tileFragmentShader)
	if err != nil {
		log.Fatalln("failed to create program:", err)
	}
	gl.UseProgram(w.tileProgram)
	if err != nil {
		log.Fatalln("failed to use program:", err)
	}

	w.tileProjectionLocation = gl.GetUniformLocation(w.tileProgram, gl.Str("projectionMatrix\x00"))
	if w.tileProjectionLocation == -1 {
		log.Fatalln("cannot get location of tex projectionMatrix")
	}

	w.tileModelLocation = gl.GetUniformLocation(w.tileProgram, gl.Str("modelMatrix\x00"))
	if w.tileModelLocation == -1 {
		log.Fatalln("cannot get location of tex modelMatrix")
	}
	w.texLocation = gl.GetUniformLocation(w.tileProgram, gl.Str("tex\x00"))
	if w.texLocation == -1 {
		log.Fatalln("cannot get location of tex")
	}
	w.texPositionLocation = gl.GetAttribLocation(w.tileProgram, gl.Str("position\x00"))
	if w.texPositionLocation == -1 {
		log.Fatalln("cannot get location of tex position")
	}
	w.texCoordLocation = gl.GetAttribLocation(w.tileProgram, gl.Str("texCoord_buffer\x00"))
	if w.texCoordLocation == -1 {
		log.Fatalln("cannot get location of texCoord_buffer")
	}
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr on Get texCoord_buffer location", err)
	}
	gl.Uniform1i(w.texLocation, 0)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after Uniform1i", err)
	}
	w.projectionMatrix = mgl32.Ortho2D(0, 780, 580, 0)

	gl.GenBuffers(1, &w.texVBO)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after GetBuffers 1", err)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, w.texVBO)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after BindBuffer 1", err)
	}
	gl.BufferData(gl.ARRAY_BUFFER, len(texCoords)*4, gl.Ptr(texCoords), gl.STATIC_DRAW)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after BufferData 1", err)
	}

	gl.GenBuffers(1, &w.tileVerticesVBO)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after GetBuffers 2", err)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, w.tileVerticesVBO)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after BindBuffer 2", err)
	}
	gl.BufferData(gl.ARRAY_BUFFER, len(tileVertices)*4, gl.Ptr(tileVertices), gl.STATIC_DRAW)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after BufferData 2", err)
	}

	/*	gl.GenBuffers(1, &w.portalVerticesVBO)
		if err := gl.GetError(); err != 0 {
			log.Fatalln("erorr after GetBuffers 3", err)
		}
		gl.BindBuffer(gl.ARRAY_BUFFER, w.portalVerticesVBO)
		if err := gl.GetError(); err != 0 {
			log.Fatalln("erorr after BindBuffer 3", err)
		}
		gl.BufferData(gl.ARRAY_BUFFER, len(portalVertices)*4, gl.Ptr(portalVertices), gl.STATIC_DRAW)
		if err := gl.GetError(); err != 0 {
			log.Fatalln("erorr after BufferData 3", err)
		}

		w.portalTex = createPortalTexture(portalRadius)
		if err := gl.GetError(); err != 0 {
			log.Fatalln("erorr after createPortalTexture", err)
		}*/
	w.meshProgram, err = newProgram(meshVertexShader, meshFragmentShader)
	if err != nil {
		log.Fatalln("failed to create mesh program:", err)
	}
	gl.UseProgram(w.meshProgram)
	if err != nil {
		log.Fatalln("failed to use program:", err)
	}
	w.meshProjectionLocation = gl.GetUniformLocation(w.meshProgram, gl.Str("projectionMatrix\x00"))
	if w.meshProjectionLocation == -1 {
		log.Fatalln("cannot get location of projectionMatrix")
	}

	w.meshModelLocation = gl.GetUniformLocation(w.meshProgram, gl.Str("modelMatrix\x00"))
	if w.meshModelLocation == -1 {
		log.Fatalln("cannot get location of modelMatrix")
	}
	w.meshPositionLocation = gl.GetAttribLocation(w.meshProgram, gl.Str("position\x00"))
	if w.meshPositionLocation == -1 {
		log.Fatalln("cannot get location of mesh position")
	}
	w.meshColorLocation = gl.GetUniformLocation(w.meshProgram, gl.Str("color\x00"))
	if w.meshColorLocation == -1 {
		log.Fatalln("cannot get location of color")
	}
	w.meshRadiusLocation = gl.GetUniformLocation(w.meshProgram, gl.Str("radius\x00"))
	if w.meshRadiusLocation == -1 {
		log.Fatalln("cannot get location of radius")
	}

	gl.GenBuffers(1, &w.portalVerticesVBO)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after GetBuffers 3", err)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, w.portalVerticesVBO)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after BindBuffer 3", err)
	}
	gl.BufferData(gl.ARRAY_BUFFER, len(portalVertices)*4, gl.Ptr(portalVertices), gl.STATIC_DRAW)
	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after BufferData 3", err)
	}

}
func (w *MapWindow) DrawAllTiles() {
	for coord, tex := range w.mapTiles {
		gl.UseProgram(w.tileProgram)
		gl.UniformMatrix4fv(w.tileProjectionLocation, 1, false, &w.projectionMatrix[0])
		dx := float64(coord.X)*256 - w.x0
		dy := float64(coord.Y)*256 - w.y0
		modelMatrix := mgl32.Translate3D(float32(dx), float32(dy), 0)

		gl.UniformMatrix4fv(w.tileModelLocation, 1, false, &modelMatrix[0])

		gl.BindBuffer(gl.ARRAY_BUFFER, w.tileVerticesVBO)
		gl.VertexAttribPointer(uint32(w.texPositionLocation), 3, gl.FLOAT, false, 0, nil)
		gl.EnableVertexAttribArray(uint32(w.texPositionLocation))

		gl.BindBuffer(gl.ARRAY_BUFFER, w.texVBO)
		gl.VertexAttribPointer(uint32(w.texCoordLocation), 2, gl.FLOAT, false, 0, nil)
		gl.EnableVertexAttribArray(uint32(w.texCoordLocation))

		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, tex)

		gl.Uniform1i(w.texLocation, 0)
		gl.DrawArrays(gl.TRIANGLE_FAN, 0, 4)
	}
}
func (w *MapWindow) DrawAllPortals() {
	for i, portal := range w.portals {
		gl.UseProgram(w.meshProgram)
		x := float32(portal.coords.X*w.zoomPow*256 - w.x0)
		y := float32(portal.coords.Y*w.zoomPow*256 - w.y0)
		gl.UniformMatrix4fv(w.meshProjectionLocation, 1, false, &w.projectionMatrix[0])
		z := 0.1
		if i%2 == 1 {
			z = -0.1
		}
		modelMatrix := mgl32.Translate3D(x-portalRadius, y-portalRadius, float32(z))
		gl.UniformMatrix4fv(w.meshModelLocation, 1, false, &modelMatrix[0])

		gl.BindBuffer(gl.ARRAY_BUFFER, w.portalVerticesVBO)
		gl.VertexAttribPointer(uint32(w.meshPositionLocation), 3, gl.FLOAT, false, 0, nil)
		gl.EnableVertexAttribArray(uint32(w.meshPositionLocation))
		gl.Uniform1f(w.meshRadiusLocation, portalRadius)
		gl.Uniform4f(w.meshColorLocation, 1.0, 0.5, 0.0, 1.0)
		gl.DrawArrays(gl.TRIANGLE_FAN, 0, int32(len(portalVertices)/3))
		/*			gl.UseProgram(w.tileProgram)
					x := float32(portal.coords.X*w.zoomPow*256 - w.x0)
					y := float32(portal.coords.Y*w.zoomPow*256 - w.y0)
					gl.UniformMatrix4fv(w.tileProjectionLocation, 1, false, &w.projectionMatrix[0])
					modelMatrix := mgl32.Translate3D(x-portalRadius, y-portalRadius, 0)

					gl.UniformMatrix4fv(w.tileModelLocation, 1, false, &modelMatrix[0])

					gl.BindBuffer(gl.ARRAY_BUFFER, w.portalVerticesVBO)
					gl.VertexAttribPointer(uint32(w.texPositionLocation), 3, gl.FLOAT, false, 0, nil)
					gl.EnableVertexAttribArray(uint32(w.texPositionLocation))

					gl.BindBuffer(gl.ARRAY_BUFFER, w.texVBO)
					gl.VertexAttribPointer(uint32(w.texCoordLocation), 2, gl.FLOAT, false, 0, nil)
					gl.EnableVertexAttribArray(uint32(w.texCoordLocation))

					gl.ActiveTexture(gl.TEXTURE0)
					gl.BindTexture(gl.TEXTURE_2D, w.portalTex)
					gl.Uniform1i(w.texLocation, 0)
					gl.DrawArrays(gl.TRIANGLE_FAN, 0, 4)*/
	}
	gl.UseProgram(0)
	gl.DisableVertexAttribArray(uint32(w.texPositionLocation))
	gl.DisableVertexAttribArray(uint32(w.texCoordLocation))

}
func (w *MapWindow) DrawAllTilesImgui() {
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
func (w *MapWindow) DrawAllPortalsImgui() {
	imgui.NewFrame()
	orange := imgui.Packed(color.NRGBA{255, 127, 0, 255})
	black := imgui.Packed(color.NRGBA{0, 0, 0, 255})
	drawList := imgui.BackgroundDrawList()
	for i, portal := range w.portals {
		x := float32(portal.coords.X*w.zoomPow*256 - w.x0)
		y := float32(portal.coords.Y*w.zoomPow*256 - w.y0)
		drawList.AddCircleV(imgui.Vec2{x, y}, 5.2, black, 0, 0.1)
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
func (w *MapWindow) DrawAllPathsImgui() {
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
func (w *MapWindow) SetPortals(portals []lib.Portal) {
	go func() {
		for _, callback := range w.onMapChangedCallbacks {
			callback()
		}
		w.setPortalsChannel <- portals
	}()
}
func (w *MapWindow) SetPaths(paths [][]lib.Portal) {
	go func() {
		for _, callback := range w.onMapChangedCallbacks {
			callback()
		}
		w.setPathsChannel <- paths
	}()
}
func (w *MapWindow) onTileRead(coord osm.TileCoord, img image.Image) {
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
func (w *MapWindow) redrawTiles() {
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
func (w *MapWindow) showTile(coord osm.TileCoord, img image.Image) {
	if tex, ok := w.mapTiles[coord]; ok {
		deleteTexture(tex)
	}
	w.mapTiles[coord] = newTexture(img)
	for _, callback := range w.onMapChangedCallbacks {
		callback()
	}
}

func (w *MapWindow) fetchTile(coord osm.TileCoord) {
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

func (w *MapWindow) tryShowTile(coord osm.TileCoord) {
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

func createPortalTexture(radius int) uint32 {
	img := image.NewRGBA(image.Rect(0, 0, 2*radius+1, 2*radius+1))
	radiusSq := radius * radius
	radiusm2Sq := (radius - 1) * (radius - 1)
	for x := 0; x < 2*radius+1; x++ {
		for y := 0; y < 2*radius+1; y++ {
			dx, dy := x-radius, y-radius
			distSq := dx*dx + dy*dy
			if distSq > radiusSq {
				img.Set(x, y, color.NRGBA{0, 0, 0, 0})
			} else if distSq >= radiusm2Sq {
				img.Set(x, y, color.NRGBA{0, 0, 0, 255})
			} else {
				img.Set(x, y, color.NRGBA{255, 165, 0, 255})
			}
		}
	}
	return newTexture(img)
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

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.DeleteShader(vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.DeleteShader(fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	if err := gl.GetError(); err != 0 {
		log.Fatalln("erorr after creating program", err)
	}

	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
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

const portalRadius = 5

func createPortalVertices2(radius float32) []float32 {
	size := 2*radius + 1
	return []float32{
		0.0, 0.0, 0.0,
		0.0, size, 0.0,
		size, size, 0.0,
		size, 0.0, 0.0,
	}
}

var portalVertices = createPortalVertices(portalRadius)

func createPortalVertices(radius float64) []float32 {
	const NUM_DIAMETER_VERTICES = 8
	f := make([]float32, 0, 3+3*NUM_DIAMETER_VERTICES)
	f = append(f, 0.0, 0.0, 0.0)
	for i := 0; i <= NUM_DIAMETER_VERTICES; i++ {
		f = append(f,
			float32(radius*math.Sin(2.*math.Pi*float64(i)/NUM_DIAMETER_VERTICES)),
			float32(radius*math.Cos(2.*math.Pi*float64(i)/NUM_DIAMETER_VERTICES)),
			0.)
	}
	return f
}

var tileVertexShader = `
#version 120
uniform mat4 projectionMatrix;
uniform mat4 modelMatrix;
attribute vec3 position;
attribute vec2 texCoord_buffer;
varying vec2 texCoord;
void main(void) {
    gl_Position = projectionMatrix * modelMatrix * vec4(position, 1.0);
    texCoord = texCoord_buffer;
}
` + "\x00"

var tileFragmentShader = `
#version 120
uniform sampler2D tex;
varying vec2 texCoord;
void main(void) {
    gl_FragColor = texture2D(tex, texCoord);
}
` + "\x00"

var meshVertexShader = `
#version 120
uniform mat4 projectionMatrix;
uniform mat4 modelMatrix;
attribute vec3 position;
varying vec3 pos;
void main(void) {
    pos = position;
    gl_Position = projectionMatrix * modelMatrix * vec4(position, 1.0);
}
` + "\x00"

var meshFragmentShader = `
#version 120
uniform vec4 color;
uniform float radius;
varying vec3 pos;
void main(void) {
    vec4 c = step(1.0, radius - length(vec3(pos.x, pos.y, 0.0))) * color;
    c[3] = color[3];
    gl_FragColor = c;
}
` + "\x00"
