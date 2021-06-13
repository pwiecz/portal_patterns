package main

import (
	"fmt"
	"image/color"

	"github.com/golang/geo/s2"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/lib"
)

type herringboneTab struct {
	*baseTab
	b0, b1       lib.Portal
	spine        []lib.Portal
	solutionText string
	basePortals  map[string]struct{}
}

var _ pattern = (*herringboneTab)(nil)

func newHerringboneTab(portals *Portals) *herringboneTab {
	t := &herringboneTab{}
	t.baseTab = newBaseTab("Herringbone", portals, t)
	t.basePortals = make(map[string]struct{})
	t.End()

	return t
}

func (t *herringboneTab) onReset() {
	t.basePortals = make(map[string]struct{})
	t.spine = nil
	t.solutionText = ""
}
func (t *herringboneTab) onSearch(progressFunc func(int, int), onSearchDone func()) {
	portals := t.enabledPortals()
	base := []int{}
	for i, portal := range portals {
		if _, ok := t.basePortals[portal.Guid]; ok {
			base = append(base, i)
		}
	}
	go func() {
		b0, b1, spine := lib.LargestHerringbone(portals, base, 8, progressFunc)
		fltk.Awake(func() {
			t.b0, t.b1, t.spine = b0, b1, spine
			t.solutionText = fmt.Sprintf("Solution length: %d", len(t.spine))
			onSearchDone()
		})
	}()
}

func (t *herringboneTab) hasSolution() bool {
	return len(t.spine) > 0
}
func (t *herringboneTab) solutionInfoString() string {
	return t.solutionText
}
func (t *herringboneTab) solutionDrawToolsString() string {
	return lib.HerringboneDrawToolsString(t.b0, t.b1, t.spine)
}
func (t *herringboneTab) solutionPaths() [][]s2.Point {
	return [][]s2.Point{portalsToPoints(lib.HerringbonePolyline(t.b0, t.b1, t.spine))}
}

func (t *herringboneTab) portalLabel(guid string) string {
	if _, ok := t.basePortals[guid]; ok {
		return "Base"
	}
	return t.baseTab.portalLabel(guid)
}
func (t *herringboneTab) portalColor(guid string) (color.Color, color.Color) {
	if _, ok := t.basePortals[guid]; ok {
		return color.NRGBA{0, 128, 0, 128}, t.baseTab.strokeColor(guid)
	}
	return t.baseTab.portalColor(guid)
}

func (t *herringboneTab) enableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		delete(t.basePortals, guid)
	}
}

func (t *herringboneTab) disableSelectedPortals() {
	for guid := range t.portals.selectedPortals {
		t.portals.disabledPortals[guid] = struct{}{}
		delete(t.basePortals, guid)
	}
}

func (t *herringboneTab) makeSelectedPortalsBase() {
	for guid := range t.portals.selectedPortals {
		delete(t.portals.disabledPortals, guid)
		t.basePortals[guid] = struct{}{}
	}
}
func (t *herringboneTab) unmakeSelectedPortalsBase() {
	for guid := range t.portals.selectedPortals {
		delete(t.basePortals, guid)
	}
}

func (t *herringboneTab) contextMenu() *menu {
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

type herringboneState struct {
	BasePortals  []string `json:"basePortals"`
	B0           string   `json:"b0"`
	B1           string   `json:"b1"`
	Spine        []string `json:"spine"`
	SolutionText string   `json:"solutionText"`
}

func (t *herringboneTab) state() herringboneState {
	state := herringboneState{
		B0:           t.b0.Guid,
		B1:           t.b1.Guid,
		SolutionText: t.solutionText,
	}
	for baseGUID := range t.basePortals {
		state.BasePortals = append(state.BasePortals, baseGUID)
	}
	for _, spinePortal := range t.spine {
		state.Spine = append(state.Spine, spinePortal.Guid)
	}
	return state
}

func (t *herringboneTab) load(state herringboneState) error {
	t.basePortals = make(map[string]struct{})
	for _, baseGUID := range state.BasePortals {
		if _, ok := t.portals.portalMap[baseGUID]; !ok {
			return fmt.Errorf("unknown herringbone base portal %s", baseGUID)
		}
		t.basePortals[baseGUID] = struct{}{}
	}
	if b0Portal, ok := t.portals.portalMap[state.B0]; !ok && state.B0 != "" {
		return fmt.Errorf("unknown herringbone.b0 portal %s", state.B0)
	} else {
		t.b0 = b0Portal
	}
	if b1Portal, ok := t.portals.portalMap[state.B1]; !ok && state.B1 != "" {
		return fmt.Errorf("unknown herringbone.b1 portal %s", state.B1)
	} else {
		t.b1 = b1Portal
	}
	t.spine = nil
	for _, spineGUID := range state.Spine {
		if spinePortal, ok := t.portals.portalMap[spineGUID]; !ok {
			return fmt.Errorf("unknown herringbone spine portal %s", spineGUID)
		} else {
			t.spine = append(t.spine, spinePortal)
		}
	}
	t.solutionText = state.SolutionText
	return nil
}
