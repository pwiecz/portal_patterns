package main

import "log"
import "math"
import "image"
import "time"

import "github.com/golang/geo/r2"
import "github.com/golang/geo/s2"
import "github.com/pwiecz/atk/tk"
import "github.com/pwiecz/portal_patterns/lib"

var projection = NewWebMercatorProjection()

type tile struct {
	x, y, zoom int
	image      image.Image
}
type empty struct{}

type mapPortal struct {
	coords r2.Point
	name   string
	shape  *tk.CanvasOval
}

type SolutionMap struct {
	*tk.Window
	font               *tk.SysFont
	fontDescription    string
	textAscent         int
	textDescent        int
	layout             *tk.PackLayout
	canvas             *tk.Canvas
	nameLabel          *tk.CanvasText
	nameLabelBackgound *tk.CanvasRectangle
	zoom               int
	zoomPow            float64
	x0, y0             float64
	dragPosX, dragPosY int
	portalPaths        [][]r2.Point
	lines              []*tk.CanvasLine
	tileCache          *MapTiles
	mapTiles           map[tileCoord]*tk.CanvasImage
	missingTiles       map[tileCoord]bool
	portals            map[string]mapPortal
	onPortalLeftClick  func(string)
	onPortalRightClick func(string, int, int)
}

func NewSolutionMap(parent tk.Widget, title string) *SolutionMap {
	s := &SolutionMap{}
	s.Window = tk.NewWindow()
	s.Window.SetTitle(title + " - © OpenStreetMap")
	s.layout = tk.NewVPackLayout(s.Window)
	s.canvas = tk.NewCanvas(s.Window, tk.CanvasAttrBackground("#C8C8C8"))
	s.layout.AddWidgetEx(s.canvas, tk.FillBoth, true, tk.AnchorNorth)
	s.Window.ResizeN(800, 600)
	s.canvas.BindEvent("<Configure>", func(e *tk.Event) {
		s.canvas.SetWidth(e.Width)
		s.canvas.SetHeight(e.Height)
		s.showTiles()
	})
	s.canvas.SetFocus()
	s.canvas.BindEvent("<Button1-Motion>", func(e *tk.Event) { s.OnDrag(e) })
	//s.canvas.BindEvent("<Control-Button-1>", func(e *tk.Event) { fmt.Println("Ctrl-Click") })
	s.canvas.BindEvent("<ButtonPress-1>", func(e *tk.Event) { s.OnButtonPress(e) })
	s.canvas.BindEvent("<ButtonPress-4>", func(e *tk.Event) { s.OnScrollUp(e) })
	s.canvas.BindEvent("<ButtonPress-5>", func(e *tk.Event) { s.OnScrollDown(e) })
	s.canvas.BindEvent("<MouseWheel>", func(e *tk.Event) {
		if e.WheelDelta < 0 {
			s.OnScrollDown(e)
		} else {
			s.OnScrollUp(e)
		}
	})
	s.canvas.BindEvent("<KeyRelease-plus>", func(e *tk.Event) {
		s.OnZoomIn(s.canvas.Width()/2, s.canvas.Height()/2)
	})
	s.canvas.BindEvent("<KeyRelease-minus>", func(e *tk.Event) {
		s.OnZoomOut(s.canvas.Width()/2, s.canvas.Height()/2)
	})
	s.portals = make(map[string]mapPortal)
	s.mapTiles = make(map[tileCoord]*tk.CanvasImage)
	s.missingTiles = make(map[tileCoord]bool)
	s.tileCache = NewMapTiles()
	s.tileCache.SetOnTileRead(func(coord tileCoord, tile *tk.Image) { s.onTileRead(coord, tile) })
	s.font = tk.LoadSysFont(tk.SysTextFont)
	s.fontDescription = s.font.Description()
	s.textAscent = s.font.Ascent()
	s.textDescent = s.font.Descent()
	s.layout.Repack()
	return s
}

func (s *SolutionMap) SetPortalColor(guid, color string) {
	if portal, ok := s.portals[guid]; ok {
		portal.shape.SetFill(color)
	}
}
func (s *SolutionMap) RaisePortal(guid string) {
	if portal, ok := s.portals[guid]; ok {
		portal.shape.Raise()
	}
}
func (s *SolutionMap) Clear() {
	s.canvas.DeleteAllItems()
	s.portalPaths = nil
	s.lines = nil
	s.mapTiles = make(map[tileCoord]*tk.CanvasImage)
	s.missingTiles = make(map[tileCoord]bool)
	s.portals = make(map[string]mapPortal)
}
func (s *SolutionMap) OnDrag(e *tk.Event) {
	dx, dy := float64(e.PosX-s.dragPosX), float64(e.PosY-s.dragPosY)
	if dx == 0. && dy == 0. {
		return
	}
	s.dragPosX, s.dragPosY = e.PosX, e.PosY
	s.shiftMap(-dx, -dy)
}
func (s *SolutionMap) shiftMap(dx, dy float64) {
	s.x0 += dx
	s.y0 += dy
	for _, portal := range s.portals {
		portal.shape.Move(-dx, -dy)
	}
	for _, line := range s.lines {
		line.Move(-dx, -dy)
	}
	for _, tileImage := range s.mapTiles {
		if tileImage != nil {
			tileImage.Move(-dx, -dy)
		}
	}
	s.showTiles()
}
func (s *SolutionMap) OnButtonPress(e *tk.Event) {
	s.dragPosX, s.dragPosY = e.PosX, e.PosY
}
func (s *SolutionMap) OnZoomIn(cx, cy int) {
	if s.zoom >= 19 {
		return
	}
	s.zoom++
	s.zoomPow *= 2.0
	s.x0 = (s.x0+float64(cx))*2.0 - float64(cx)
	s.y0 = (s.y0+float64(cy))*2.0 - float64(cy)
	s.showTiles()
	s.showSolution()
	s.setItemCoords()
	if len(s.mapTiles) > 0 {
		s.canvas.LowerItems("tile")
	}
	if len(s.lines) > 0 {
		if len(s.mapTiles) > 0 {
			s.canvas.RaiseItemsAbove("link", "tile")
		}
		if len(s.portals) > 0 {
			s.canvas.RaiseItemsAbove("portal", "link")
		}
	}
}
func (s *SolutionMap) OnZoomOut(cx, cy int) {
	if s.zoom <= 0 {
		return
	}
	s.zoom--
	s.zoomPow *= 0.5
	s.x0 = (s.x0+float64(cx))*0.5 - float64(cx)
	s.y0 = (s.y0+float64(cy))*0.5 - float64(cy)
	s.showTiles()
	s.showSolution()
	s.setItemCoords()
	if len(s.mapTiles) > 0 {
		s.canvas.LowerItems("tile")
	}
	if len(s.lines) > 0 {
		if len(s.mapTiles) > 0 {
			s.canvas.RaiseItemsAbove("link", "tile")
		}
		if len(s.portals) > 0 {
			s.canvas.RaiseItemsAbove("portal", "link")
		}
	}
}

func (s *SolutionMap) showTile(coord tileCoord, tileImage *tk.Image) {
	if tile, ok := s.mapTiles[coord]; ok {
		s.canvas.DeleteImage(tile)
	}
	dx := float64(coord.x)*256.0 - s.x0
	dy := float64(coord.y)*256.0 - s.y0
	mapTile := s.canvas.CreateImage(dx, dy, tk.CanvasItemAttrImage(tileImage), tk.CanvasItemAttrAnchor(tk.AnchorNorthWest), tk.CanvasItemAttrTags([]string{"tile"}))
	mapTile.Lower()
	s.mapTiles[coord] = mapTile
}

func (s *SolutionMap) onTileRead(coord tileCoord, tileImage *tk.Image) {
	if s.missingTiles[coord] {
		if tileImage == nil {
			// try fetching again after 1 second
			timer := time.NewTimer(time.Second)
			go func() {
				<-timer.C
				tk.Async(func(){
					// check if we still need the tile before refetching
					if s.missingTiles[coord] {
						s.tryShowTile(coord)
					}
				})
			}()
			return
		}
		s.showTile(coord, tileImage)
		delete(s.missingTiles, coord)
	}
}
func (s *SolutionMap) showTiles() {
	if s.zoomPow == 0 {
		return
	}
	tileCoords := make(map[tileCoord]bool)
	maxCoord := 1 << s.zoom
	for x := int(math.Floor(s.x0 / 256)); x <= int(math.Floor(s.x0+float64(s.canvas.Width())))/256; x++ {
		for y := int(math.Floor(s.y0 / 256)); y <= int(math.Floor(s.y0+float64(s.canvas.Height())))/256; y++ {
			if y >= 0 && y < maxCoord {
				tileCoords[tileCoord{x, y, s.zoom}] = true
			}
		}
	}
	for coord, image := range s.mapTiles {
		if _, ok := tileCoords[coord]; !ok {
			s.canvas.DeleteImage(image)
			delete(s.mapTiles, coord)
		} else {
			delete(tileCoords, coord)
		}
	}
	s.missingTiles = make(map[tileCoord]bool)
	for coord := range tileCoords {
		s.tryShowTile(coord)
	}
}
func (s *SolutionMap) tryShowTile(coord tileCoord) {
	tileImage, ok := s.tileCache.GetTile(coord)
	if tileImage != nil {
		s.showTile(coord, tileImage)
	}
	if !ok {
		s.missingTiles[coord] = true
	}
}
func (s *SolutionMap) showSolution() {
	for _, line := range s.lines {
		s.canvas.DeleteLine(line)
	}
	s.lines = []*tk.CanvasLine{}
	for _, path := range s.portalPaths {
		coords := make([]tk.CanvasCoordinates, 0, len(path))
		for _, point := range path {
			x, y := s.GeoToScreenCoordinates(point.X, point.Y)
			coords = append(coords, tk.CanvasPixelCoords(x, y))
		}
		line := s.canvas.CreateLine(coords, tk.CanvasItemAttrWidth(1), tk.CanvasItemAttrFill("blue"), tk.CanvasItemAttrTags([]string{"link"}))
		s.lines = append(s.lines, line)
	}

}
func (s *SolutionMap) setItemCoords() {
	for _, portal := range s.portals {
		x, y := s.GeoToScreenCoordinates(portal.coords.X, portal.coords.Y)
		err := portal.shape.MoveTo(x-4, y-4)
		if err != nil {
			log.Println(err)
		}
	}
}
func (s *SolutionMap) GeoToScreenCoordinates(x, y float64) (float64, float64) {
	return x*s.zoomPow*256.0 - s.x0, y*s.zoomPow*256.0 - s.y0

}
func (s *SolutionMap) ScrollToPortal(guid string) {
	portal, ok := s.portals[guid]
	if !ok {
		log.Println("Cannot locate portal", guid, "on the map")
		return
	}
	x, y := s.GeoToScreenCoordinates(portal.coords.X, portal.coords.Y)
	if x >= 0 && x < float64(s.canvas.Width()) &&
		y >= 0 && y < float64(s.canvas.Height()) {
		return
	}
	s.shiftMap(x-float64(s.canvas.Width())/2, y-float64(s.canvas.Height())/2)
}
func (s *SolutionMap) OnScrollUp(e *tk.Event) {
	s.OnZoomIn(e.PosX, e.PosY)
}
func (s *SolutionMap) OnScrollDown(e *tk.Event) {
	s.OnZoomOut(e.PosX, e.PosY)
}
func (s *SolutionMap) OnPortalLeftClick(onPortalLeftClick func(string)) {
	s.onPortalLeftClick = onPortalLeftClick
}
func (s *SolutionMap) OnPortalRightClick(onPortalRightClick func(string, int, int)) {
	s.onPortalRightClick = onPortalRightClick
}
func (s *SolutionMap) SetPortals(portals map[string]lib.Portal) {
	for _, portal := range s.portals {
		s.canvas.DeleteOval(portal.shape)
	}
	s.portals = make(map[string]mapPortal)
	if len(portals) == 0 {
		return
	}
	chQuery := s2.NewConvexHullQuery()
	for _, portal := range portals {
		chQuery.AddPoint(s2.PointFromLatLng(portal.LatLng))
	}
	chQuery.ConvexHull()

	minX, minY, maxX, maxY := math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for _, portal := range portals {
		mapCoords := projection.FromLatLng(portal.LatLng)
		minX = math.Min(mapCoords.X, minX)
		minY = math.Min(mapCoords.Y, minY)
		maxX = math.Max(mapCoords.X, maxX)
		maxY = math.Max(mapCoords.Y, maxY)
	}
	numTilesX := math.Ceil(float64(s.canvas.Width()) / 256.)
	numTilesY := math.Ceil(float64(s.canvas.Height()) / 256.)
	for s.zoom = 19; s.zoom >= 0; s.zoom-- {
		zoomPow := math.Pow(2., float64(s.zoom))
		minXTile, minYTile := math.Floor(minX*zoomPow), math.Floor(minY*zoomPow)
		maxXTile, maxYTile := math.Floor(maxX*zoomPow), math.Floor(maxY*zoomPow)
		if maxXTile-minXTile+1 <= numTilesX && maxYTile-minYTile+1 <= numTilesY {
			break
		}
	}
	if s.zoom < 0 {
		s.zoom = 0
	}
	s.zoomPow = math.Pow(2., float64(s.zoom))
	s.x0 = (maxX+minX)*s.zoomPow*0.5*256.0 - float64(s.canvas.Width())*0.5
	s.y0 = (maxY+minY)*s.zoomPow*0.5*256.0 - float64(s.canvas.Height())*0.5
	s.showTiles()
	for _, portal := range portals {
		mapCoords := projection.FromLatLng(portal.LatLng)
		x, y := s.GeoToScreenCoordinates(mapCoords.X, mapCoords.Y)
		item := s.canvas.CreateOval(x-4, y-4, x+5, y+5, tk.CanvasItemAttrFill("orange"), tk.CanvasItemAttrTags([]string{"portal"}))
		item.Raise()
		guid := portal.Guid // local copy to make closure captures work correctly
		s.portals[guid] = mapPortal{coords: mapCoords, name: portal.Name, shape: item}
		item.BindEvent("<Button-1>", func(e *tk.Event) {
			if s.onPortalLeftClick != nil {
				s.onPortalLeftClick(guid)
			}
		})
		item.BindEvent("<Button-3>", func(e *tk.Event) {
			if s.onPortalRightClick != nil {
				s.onPortalRightClick(guid, e.GlobalPosX, e.GlobalPosY)
			}
		})
		item.BindEvent("<Enter>", func(e *tk.Event) {
			s.onPortalEntered(guid)
		})
		item.BindEvent("<Leave>", func(e *tk.Event) {
			s.onPortalLeft(guid)
		})
	}
}

func (s *SolutionMap) SetSolution(lines [][]lib.Portal) {
	tesselator := s2.NewEdgeTessellator(projection, 1e-3)
	s.portalPaths = make([][]r2.Point, 0, len(lines))
	for _, line := range lines {
		path := []r2.Point{}
		for i := 1; i < len(line); i++ {
			path = tesselator.AppendProjected(s2.PointFromLatLng(line[i-1].LatLng), s2.PointFromLatLng(line[i].LatLng), path)
		}
		s.portalPaths = append(s.portalPaths, path)
	}
	s.showSolution()
	if len(s.lines) > 0 {
		s.canvas.RaiseItemsAbove("portal", "link")
	}
}

func (s *SolutionMap) onPortalEntered(guid string) {
	if s.nameLabel != nil {
		s.canvas.DeleteText(s.nameLabel)
		s.nameLabel = nil
	}
	if s.nameLabelBackgound != nil {
		s.canvas.DeleteRectangle(s.nameLabelBackgound)
		s.nameLabelBackgound = nil
	}
	portal, ok := s.portals[guid]
	if !ok {
		return
	}
	x, y := s.GeoToScreenCoordinates(portal.coords.X, portal.coords.Y)
	backgroundWidth := float64(s.font.MeasureTextWidth(portal.name) + 6)
	backgroundHeight := float64(s.textAscent + s.textDescent + 8)
	backgroundX, backgroundY := x-backgroundWidth/2, y-9
	if backgroundX < 0 {
		backgroundX = 0
	} else if backgroundX+backgroundWidth >= float64(s.canvas.Width()) {
		backgroundX = float64(s.canvas.Width()) - backgroundWidth
	}
	if backgroundY-backgroundHeight < 0 {
		backgroundY = y + 5 + backgroundHeight + 4
	}
	s.nameLabelBackgound = s.canvas.CreateRectangle(backgroundX, backgroundY, backgroundX+backgroundWidth, backgroundY-backgroundHeight, tk.CanvasItemAttrFill("white"))
	s.nameLabel = s.canvas.CreateText(backgroundX+backgroundWidth/2, backgroundY-3, tk.CanvasItemAttrText(portal.name), tk.CanvasItemAttrFont(s.fontDescription), tk.CanvasItemAttrAnchor(tk.AnchorSouth))
}
func (s *SolutionMap) onPortalLeft(guid string) {
	if s.nameLabel != nil {
		s.canvas.DeleteText(s.nameLabel)
		s.nameLabel = nil
	}
	if s.nameLabelBackgound != nil {
		s.canvas.DeleteRectangle(s.nameLabelBackgound)
		s.nameLabelBackgound = nil
	}
}
