package main

import (
	"fmt"
	"image/color"
	"runtime"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/lib"
)

type doubleHerringboneTab struct {
	*baseTab
	b0, b1         lib.Portal
	spine0, spine1 []lib.Portal
	solutionText   string
	basePortals    map[string]struct{}
}

var _ pattern = (*doubleHerringboneTab)(nil)

func newDoubleHerringboneTab(portals *Portals) *doubleHerringboneTab {
	t := &doubleHerringboneTab{}
	t.baseTab = newBaseTab("Double Herringbone", portals, t)
	t.basePortals = make(map[string]struct{})
	t.End()

	return t
}

func (t *doubleHerringboneTab) onReset() {
	t.basePortals = make(map[string]struct{})
	t.spine0 = nil
	t.spine1 = nil
	t.solutionText = ""
}
func (t *doubleHerringboneTab) onSearch(progressFunc func(int, int), onSearchDone func()) {
	portals := t.enabledPortals()
	base := []int{}
	for i, portal := range portals {
		if _, ok := t.basePortals[portal.Guid]; ok {
			base = append(base, i)
		}
	}
	go func() {
		b0, b1, spine0, spine1 := lib.LargestDoubleHerringbone(portals, base, runtime.GOMAXPROCS(0), progressFunc)
		fltk.Awake(func() {
			t.b0, t.b1, t.spine0, t.spine1 = b0, b1, spine0, spine1
			t.solutionText = fmt.Sprintf("Solution length: %d + %d", len(t.spine0), len(t.spine1))
		})
		onSearchDone()
	}()
}
func (t *doubleHerringboneTab) hasSolution() bool {
	return len(t.spine0)+len(t.spine1) > 0
}
func (t *doubleHerringboneTab) solutionInfoString() string {
	return t.solutionText
}
func (t *doubleHerringboneTab) solutionDrawToolsString() string {
	return lib.DoubleHerringboneDrawToolsString(t.b0, t.b1, t.spine0, t.spine1)
}
func (t *doubleHerringboneTab) solutionPaths() [][]s2.Point {
	return [][]s2.Point{portalsToPoints(lib.DoubleHerringbonePolyline(t.b0, t.b1, t.spine0, t.spine1))}
}
func (t *doubleHerringboneTab) portalLabel(guid string) string {
	if _, ok := t.basePortals[guid]; ok {
		return "Base"
	}
	return t.baseTab.portalLabel(guid)
}

func (t *doubleHerringboneTab) portalColor(guid string) (color.Color, color.Color) {
	if _, ok := t.basePortals[guid]; ok {
		return color.NRGBA{0, 128, 0, 128}, t.baseTab.strokeColor(guid)
	}
	return t.baseTab.portalColor(guid)
}

func (t *doubleHerringboneTab) enableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		delete(t.basePortals, guid)
	}
}

func (t *doubleHerringboneTab) disableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		t.portals.disabledPortals[guid] = struct{}{}
		delete(t.basePortals, guid)
	}
}

func (t *doubleHerringboneTab) makeSelectedPortalsBase() {
	for guid := range t.portals.selectedPortals {
		delete(t.portals.disabledPortals, guid)
		t.basePortals[guid] = struct{}{}
	}
}
func (t *doubleHerringboneTab) unmakeSelectedPortalsBase() {
	for guid := range t.portals.selectedPortals {
		delete(t.basePortals, guid)
	}
}

func (t *doubleHerringboneTab) contextMenu() *menu {
	var aSelectedGUID string
	numSelectedEnabled := 0
	numSelectedDisabled := 0
	numSelectedBase := 0
	numSelectedNotBase := 0
	for guid := range t.portals.selectedPortals {
		aSelectedGUID = guid
		if _, ok := t.portals.disabledPortals[guid]; ok {
			numSelectedDisabled++
		} else {
			numSelectedEnabled++
		}
		if _, ok := t.basePortals[guid]; ok {
			numSelectedBase++
		} else {
			numSelectedNotBase++
		}
	}
	menu := &menu{}
	if len(t.portals.selectedPortals) > 1 {
		menu.header = fmt.Sprintf("%d portals selected", len(t.portals.selectedPortals))
	} else if len(t.portals.selectedPortals) == 1 {
		menu.header = t.portals.portalMap[aSelectedGUID].Name
	}
	if numSelectedDisabled > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Enable", t.enableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Enable All", t.enableSelectedPortals})
		}
	}
	if numSelectedEnabled > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Disable", t.disableSelectedPortals})
		} else {
			menu.items = append(menu.items, menuItem{"Disable All", t.disableSelectedPortals})
		}
	}
	if numSelectedBase > 0 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Unmake base", t.unmakeSelectedPortalsBase})
		} else {
			menu.items = append(menu.items, menuItem{"Unmake all base", t.unmakeSelectedPortalsBase})
		}
	}
	if numSelectedNotBase > 0 && numSelectedNotBase+len(t.basePortals) <= 2 {
		if len(t.portals.selectedPortals) == 1 {
			menu.items = append(menu.items, menuItem{"Make base", t.makeSelectedPortalsBase})
		} else {
			menu.items = append(menu.items, menuItem{"Make all base", t.makeSelectedPortalsBase})
		}
	}
	return menu
}

type doubleHerringboneState struct {
	BasePortals  []string `json:"basePortals"`
	B0           string   `json:"b0"`
	B1           string   `json:"b1"`
	Spine0       []string `json:"spine0"`
	Spine1       []string `json:"spine1"`
	SolutionText string   `json:"solutionText"`
}

func (t *doubleHerringboneTab) state() doubleHerringboneState {
	state := doubleHerringboneState{
		B0:           t.b0.Guid,
		B1:           t.b1.Guid,
		SolutionText: t.solutionText,
	}
	for baseGUID := range t.basePortals {
		state.BasePortals = append(state.BasePortals, baseGUID)
	}
	for _, spine0Portal := range t.spine0 {
		state.Spine0 = append(state.Spine0, spine0Portal.Guid)
	}
	for _, spine1Portal := range t.spine1 {
		state.Spine1 = append(state.Spine1, spine1Portal.Guid)
	}
	return state
}

func (t *doubleHerringboneTab) load(state doubleHerringboneState) error {
	t.basePortals = make(map[string]struct{})
	for _, baseGUID := range state.BasePortals {
		if _, ok := t.portals.portalMap[baseGUID]; !ok {
			return fmt.Errorf("unknown doubleHerringbone base portal \"%s\"", baseGUID)
		}
		t.basePortals[baseGUID] = struct{}{}
	}
	if b0Portal, ok := t.portals.portalMap[state.B0]; !ok && state.B0 != "" {
		return fmt.Errorf("unknown doubleHerringbone.b0 portal \"%s\"", state.B0)
	}
	t.b0 = b0Portal
	if b1Portal, ok := t.portals.portalMap[state.B1]; !ok && state.B1 != "" {
		return fmt.Errorf("unknown doubleHerringbone.b1 portal \"%s\"", state.B1)
	}
	t.b1 = b1Portal
	t.spine0 = nil
	for _, spine0GUID := range state.Spine0 {
		if spine0Portal, ok := t.portals.portalMap[spine0GUID]; !ok {
			return fmt.Errorf("unknown doubleHerringbone spine0 portal \"%s\"", spine0GUID)
		}
		t.spine0 = append(t.spine0, spine0Portal)
	}
	t.spine1 = nil
	for _, spine1GUID := range state.Spine1 {
		if spine1Portal, ok := t.portals.portalMap[spine1GUID]; !ok {
			return fmt.Errorf("unknown doubleHerringbone spine1 portal \"%s\"", spine1GUID)
		}
		t.spine1 = append(t.spine1, spine1Portal)
	}
	t.solutionText = state.SolutionText
	return nil
}
