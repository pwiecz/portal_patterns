package main

import "math"
import "image"

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
	shape  *tk.CanvasOval
}

type SolutionMap struct {
	*tk.Window
	layout             *tk.PackLayout
	canvas             *tk.Canvas
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
	s.Window.SetTitle(title)
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
	s.canvas.BindEvent("<ButtonPress-1>", func(e *tk.Event) { s.OnButtonPress(e) })
	s.canvas.BindEvent("<ButtonPress-4>", func(e *tk.Event) { s.OnScrollUp(e) })
	s.canvas.BindEvent("<ButtonPress-5>", func(e *tk.Event) { s.OnScrollDown(e) })
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
	s.tileCache.SetOnTileRead(func(coord tileCoord, tile image.Image) { s.onTileRead(coord, tile) })
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
	s.x0 -= dx
	s.y0 -= dy
	s.dragPosX, s.dragPosY = e.PosX, e.PosY
	for _, portal := range s.portals {
		portal.shape.Move(dx, dy)
	}
	for _, line := range s.lines {
		line.Move(dx, dy)
	}
	for _, tileImage := range s.mapTiles {
		if tileImage != nil {
			tileImage.Move(dx, dy)
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
	s.zoom += 1
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
	s.zoom -= 1
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

func (s *SolutionMap) showTile(coord tileCoord, tileImage image.Image) {
	dx := float64(coord.x)*256.0 - s.x0
	dy := float64(coord.y)*256.0 - s.y0
	tkTile := tk.NewImage()
	tkTile.SetImage(tileImage)
	mapTile := s.canvas.CreateImage(dx, dy, tk.CanvasImageAttrImage(tkTile), tk.CanvasImageAttrAnchor(tk.AnchorNorthWest), tk.CanvasItemAttrTags([]string{"tile"}))
	mapTile.Lower()
	s.mapTiles[coord] = mapTile
}

func (s *SolutionMap) onTileRead(coord tileCoord, tileImage image.Image) {
	tk.Async(func() {
		if _, ok := s.missingTiles[coord]; ok {
			s.showTile(coord, tileImage)
			delete(s.missingTiles, coord)
		}
	})
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
	for coord, _ := range tileCoords {
		if tileImage, ok := s.tileCache.GetTile(coord); ok {
			s.showTile(coord, tileImage)
		} else {
			s.missingTiles[coord] = true
		}
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
		portal.shape.MoveTo(x-5, y-5)
	}
}
func (s *SolutionMap) GeoToScreenCoordinates(x, y float64) (float64, float64) {
	return x*s.zoomPow*256.0 - s.x0, y*s.zoomPow*256.0 - s.y0

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
func (s *SolutionMap) SetPortals(portals []*HomogeneousPortal) {
	for _, portal := range s.portals {
		s.canvas.DeleteOval(portal.shape)
	}
	s.portals = make(map[string]mapPortal)
	if len(portals) == 0 {
		return
	}
	if len(portals) == 1 {
		s.zoom = 19
		s.zoomPow = math.Pow(2., 19.)
		mapCoords := projection.FromLatLng(portals[0].portal.LatLng)
		s.x0 = mapCoords.X*s.zoomPow*256. - float64(s.canvas.Width())*0.5
		s.y0 = mapCoords.Y*s.zoomPow*256. - float64(s.canvas.Height())*0.5
		s.showTiles()
		x, y := s.GeoToScreenCoordinates(mapCoords.X, mapCoords.Y)
		item := s.canvas.CreateOval(x-5, y-5, x+5, y+5, tk.CanvasItemAttrFill("orange"), tk.CanvasItemAttrTags([]string{"portal"}))
		item.Raise()
		guid := portals[0].portal.Guid // local copy to make closure captures work correctly
		s.portals[guid] = mapPortal{coords: mapCoords, shape: item}
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
		return
	}
	chQuery := s2.NewConvexHullQuery()
	for _, portal := range portals {
		chQuery.AddPoint(s2.PointFromLatLng(portal.portal.LatLng))
	}
	chQuery.ConvexHull()

	minX, minY, maxX, maxY := math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	for _, portal := range portals {
		mapCoords := projection.FromLatLng(portal.portal.LatLng)
		minX = math.Min(mapCoords.X, minX)
		minY = math.Min(mapCoords.Y, minY)
		maxX = math.Max(mapCoords.X, maxX)
		maxY = math.Max(mapCoords.Y, maxY)
	}
	numTilesX := math.Ceil(float64(s.canvas.Width()) / 256.)
	numTilesY := math.Ceil(float64(s.canvas.Height()) / 256.)
	maxZoom := -1
	for zoom := 0; zoom <= 18; zoom++ {
		zoomPow := math.Pow(2., float64(zoom))
		minXTile, minYTile := math.Floor(minX*zoomPow), math.Floor(minY*zoomPow)
		maxXTile, maxYTile := math.Floor(maxX*zoomPow), math.Floor(maxY*zoomPow)
		if maxXTile-minXTile+1 > numTilesX || maxYTile-minYTile+1 > numTilesY {
			maxZoom = zoom - 1
			break
		}
	}
	if maxZoom == -1 {
		return
	}
	s.zoom = maxZoom
	width, height := math.Min(1., (maxX-minX)*1.2), math.Min(1., (maxY-minY)*1.2)
	s.zoomPow = math.Pow(2., float64(s.zoom))
	dim := math.Max(width, height) * s.zoomPow
	s.x0 = (minX*s.zoomPow - 0.1*dim) * 256.0
	s.y0 = (minY*s.zoomPow - 0.1*dim) * 256.0
	s.showTiles()
	for _, portal := range portals {
		mapCoords := projection.FromLatLng(portal.portal.LatLng)
		x, y := s.GeoToScreenCoordinates(mapCoords.X, mapCoords.Y)
		item := s.canvas.CreateOval(x-5, y-5, x+5, y+5, tk.CanvasItemAttrFill("orange"), tk.CanvasItemAttrTags([]string{"portal"}))
		item.Raise()
		guid := portal.portal.Guid // local copy to make closure captures work correctly
		s.portals[guid] = mapPortal{coords: mapCoords, shape: item}
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

func (s *SolutionMap) GetTile(x, y, zoom int) {
}
