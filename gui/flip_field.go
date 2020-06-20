package main

import "fmt"
import "runtime"
import "strconv"

import "github.com/golang/geo/s2"
import "github.com/pwiecz/portal_patterns/lib"
import "github.com/pwiecz/atk/tk"

type flipFieldTab struct {
	*baseTab
	maxFlipPortals           *tk.Entry
	numBackbonePortals       *tk.Entry
	exactBackbonePortalLimit *tk.CheckButton
	simpleBackbone           *tk.CheckButton
	backbone                 []lib.Portal
	rest                     []lib.Portal
}

func NewFlipFieldTab(parent *Window, conf *Configuration) *flipFieldTab {
	t := &flipFieldTab{}
	t.baseTab = NewBaseTab(parent, "Flip Field", conf)

	t.AddWidget(tk.NewLabel(parent, "EXPERIMENTAL: SLOW AND INACCURATE"))

	t.pattern = t
	addResetBox := tk.NewHPackLayout(parent)
	addResetBox.AddWidget(t.add)
	addResetBox.AddWidget(t.reset)
	t.AddWidget(addResetBox)

	numBackbonePortalsBox := tk.NewHPackLayout(parent)
	numBackbonePortalsLabel := tk.NewLabel(parent, "Num backbone portals: ")
	numBackbonePortalsBox.AddWidget(numBackbonePortalsLabel)
	t.numBackbonePortals = tk.NewEntry(parent)
	t.numBackbonePortals.SetText("16")
	numBackbonePortalsBox.AddWidget(t.numBackbonePortals)
	t.exactBackbonePortalLimit = tk.NewCheckButton(parent, "Exactly")
	numBackbonePortalsBox.AddWidget(t.exactBackbonePortalLimit)
	t.AddWidget(numBackbonePortalsBox)

	t.simpleBackbone = tk.NewCheckButton(parent, "Simple backbone")

	maxFlipPortalsBox := tk.NewHPackLayout(parent)
	maxFlipPortalsLabel := tk.NewLabel(parent, "Max flip portals: ")
	maxFlipPortalsBox.AddWidget(maxFlipPortalsLabel)
	t.maxFlipPortals = tk.NewEntry(parent)
	t.maxFlipPortals.SetText("9999")
	maxFlipPortalsBox.AddWidget(t.maxFlipPortals)
	t.AddWidget(maxFlipPortalsBox)

	t.AddWidgetEx(t.simpleBackbone, tk.FillNone, true, tk.AnchorWest)
	solutionBox := tk.NewHPackLayout(parent)
	solutionBox.AddWidget(t.find)
	solutionBox.AddWidget(t.save)
	solutionBox.AddWidget(t.copy)
	solutionBox.AddWidget(t.solutionLabel)
	t.AddWidget(solutionBox)
	t.AddWidgetEx(t.progress, tk.FillBoth, true, tk.AnchorWest)
	t.AddWidgetEx(t.portalList, tk.FillBoth, true, tk.AnchorWest)

	//t.cornerPortals = make(map[string]bool)
	return t
}

func (t *flipFieldTab) onReset() {
	//t.cornerPortals = make(map[string]bool)
}

func (t *flipFieldTab) portalLabel(guid string) string {
	if t.disabledPortals[guid] {
		return "Disabled"
	}
	return "Normal"
}

func (t *flipFieldTab) portalColor(guid string) string {
	if t.disabledPortals[guid] {
		if !t.selectedPortals[guid] {
			return "gray"
		}
		return "dark gray"
	}
	if !t.selectedPortals[guid] {
		return "orange"
	}
	return "red"
}

func (t *flipFieldTab) onPortalContextMenu(guid string, x, y int) {
	menu := NewFlipFieldPortalContextMenu(tk.RootWindow(), guid, t)
	tk.PopupMenu(menu.Menu, x, y)
}

func (t *flipFieldTab) search() {
	if len(t.portals) < 3 {
		return
	}

	t.add.SetState(tk.StateDisable)
	t.reset.SetState(tk.StateDisable)
	t.find.SetState(tk.StateDisable)
	t.save.SetState(tk.StateDisable)
	t.copy.SetState(tk.StateDisable)
	tk.Update()
	portals := []lib.Portal{}
	for _, portal := range t.portals {
		if !t.disabledPortals[portal.Guid] {
			portals = append(portals, portal)
		}
	}
	maxFlipPortals, err := strconv.Atoi(t.maxFlipPortals.Text())
	if err != nil || maxFlipPortals < 1 {
		return
	}
	numBackbonePortals, err := strconv.Atoi(t.numBackbonePortals.Text())
	if err != nil || numBackbonePortals < 1 {
		return
	}
	backbonePortalLimit := lib.FlipFieldBackbonePortalLimit{Value: numBackbonePortals}
	if t.exactBackbonePortalLimit.IsChecked() {
		backbonePortalLimit.LimitType = lib.EQUAL
	} else {
		backbonePortalLimit.LimitType = lib.LESS_EQUAL
	}
	options := []lib.FlipFieldOption{
		lib.FlipFieldProgressFunc(
			func(val int, max int) { t.onProgress(val, max) }),
		backbonePortalLimit,
		lib.FlipFieldSimpleBackbone(t.simpleBackbone.IsChecked()),
		lib.FlipFieldMaxFlipPortals(maxFlipPortals),
		lib.FlipFieldNumWorkers(runtime.GOMAXPROCS(0)),
	}
	t.backbone, t.rest = lib.LargestFlipField(portals, options...)
	hull := s2.NewConvexHullQuery()
	for _, p := range t.rest {
		hull.AddPoint(s2.PointFromLatLng(p.LatLng))
	}
	hullPoints := hull.ConvexHull().Vertices()
	if len(hullPoints) > 0 {
		hullPoints = append(hullPoints, hullPoints[0])
	}

	if t.solutionMap != nil {
		t.solutionMap.SetSolutionPoints([][]s2.Point{portalsToPoints(t.backbone), hullPoints})
	}

	solutionText := fmt.Sprintf("Num backbone portals: %d, num flip portals: %d", len(t.backbone), len(t.rest))
	t.solutionLabel.SetText(solutionText)
	t.add.SetState(tk.StateNormal)
	t.reset.SetState(tk.StateNormal)
	t.find.SetState(tk.StateNormal)
	t.save.SetState(tk.StateNormal)
	t.copy.SetState(tk.StateNormal)
	tk.Update()
}

func (t *flipFieldTab) solutionString() string {
	return fmt.Sprintf("\n[%s,%s]\n", lib.PolylineFromPortalList(t.backbone), lib.MarkersFromPortalList(t.rest))
}
func (t *flipFieldTab) EnablePortal(guid string) {
	delete(t.disabledPortals, guid)
	t.portalStateChanged(guid)
}
func (t *flipFieldTab) DisablePortal(guid string) {
	t.disabledPortals[guid] = true
	t.portalStateChanged(guid)
}

type flipFieldPortalContextMenu struct {
	*tk.Menu
}

func NewFlipFieldPortalContextMenu(parent *tk.Window, guid string, t *flipFieldTab) *flipFieldPortalContextMenu {
	l := &flipFieldPortalContextMenu{}
	l.Menu = tk.NewMenu(parent)
	if t.disabledPortals[guid] {
		enableAction := tk.NewAction("Enable")
		enableAction.OnCommand(func() { t.EnablePortal(guid) })
		l.AddAction(enableAction)
	} else {
		disableAction := tk.NewAction("Disable")
		disableAction.OnCommand(func() { t.DisablePortal(guid) })
		l.AddAction(disableAction)
	}
	return l
}
