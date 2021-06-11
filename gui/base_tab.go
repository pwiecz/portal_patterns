package main

import (
	"image/color"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/lib"
)

type baseTab struct {
	*fltk.Pack
	portals   *Portals
	portalMap map[string]lib.Portal
	pattern   pattern
}

func newBaseTab(name string, portals *Portals, pattern pattern) *baseTab {
	t := &baseTab{}
	t.Pack = fltk.NewPack(20, 40, 760, 540, name)
	t.SetType(fltk.VERTICAL)
	t.SetSpacing(5)

	t.portals = portals
	t.pattern = pattern

	fltk.NewBox(fltk.NO_BOX, 0, 0, 760, 5) // padding at the top

	return t
}

func stringSetsAreTheSame(map1 map[string]struct{}, map2 map[string]struct{}) bool {
	for s := range map1 {
		if _, ok := map2[s]; !ok {
			return false
		}
	}
	for s := range map2 {
		if _, ok := map1[s]; !ok {
			return false
		}
	}
	return true
}
func stringSetCopy(set map[string]struct{}) map[string]struct{} {
	setCopy := make(map[string]struct{})
	for s := range set {
		setCopy[s] = struct{}{}
	}
	return setCopy
}

func (t *baseTab) strokeColor(guid string) color.Color {
	if _, ok := t.portals.selectedPortals[guid]; ok {
		return color.NRGBA{0, 0, 0, 255}
	} else {
		return color.NRGBA{0, 0, 0, 0}
	}
}

func (t *baseTab) portalColor(guid string) (color.Color, color.Color) {
	if _, ok := t.portals.disabledPortals[guid]; ok {
		return color.NRGBA{128, 128, 128, 128}, t.strokeColor(guid)
	}
	return color.NRGBA{255, 128, 0, 128}, t.strokeColor(guid)
}

func (t *baseTab) portalLabel(guid string) string {
	_, isDisabled := t.portals.disabledPortals[guid]
	if isDisabled {
		return "Disabled"
	}
	return "Normal"
}

/*func (t *baseTab) OnPortalSelected(guid string) {
	selection := make(map[string]struct{})
	selection[guid] = struct{}{}
	t.OnSelectionChanged(selection)
	if t.portalList != nil {
		//t.portalList.ScrollToPortal(guid)
	}
	if t.mapWindow != nil {
		//t.solutionMap.ScrollToPortal(guid)
	}
}*/

func (t *baseTab) enabledPortals() []lib.Portal {
	portals := []lib.Portal{}
	for _, portal := range t.portals.portals {
		if _, ok := t.portals.disabledPortals[portal.Guid]; !ok {
			portals = append(portals, portal)
		}
	}
	return portals
}

func (t *baseTab) onContextMenu(x, y int) {
	menu := t.pattern.contextMenu()
	if menu == nil || len(menu.items) == 0 {
		return
	}
	mb := fltk.NewMenuButton(x, y, 100, 100, menu.header)
	//	mb.SetCallback(func() { fmt.Println("menu callback") })
	mb.SetType(fltk.POPUP3)
	for _, item := range menu.items {
		mb.Add(item.label, item.callback)
	}
	mb.Popup()
	mb.Destroy()
}

func portalsToPoints(portals []lib.Portal) []s2.Point {
	points := make([]s2.Point, 0, len(portals))
	for _, portal := range portals {
		points = append(points, s2.PointFromLatLng(portal.LatLng))
	}
	return points

}
func portalPathsToPointPaths(portalPaths [][]lib.Portal) [][]s2.Point {
	pointPaths := make([][]s2.Point, 0, len(portalPaths))
	for _, portalPath := range portalPaths {
		pointPaths = append(pointPaths, portalsToPoints(portalPath))
	}
	return pointPaths

}
