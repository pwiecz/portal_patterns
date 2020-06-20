package main

import "fmt"
import "math/rand"
import "strconv"
import "time"

import "github.com/pwiecz/portal_patterns/lib"
import "github.com/pwiecz/atk/tk"

type homogeneousTab struct {
	*baseTab
	maxDepth      *tk.Entry
	innerPortals  *tk.ComboBox
	perfect       *tk.CheckButton
	strategy      *tk.ComboBox
	solution      []lib.Portal
	depth         uint16
	anchorPortals map[string]bool
}

func NewHomogeneousTab(parent tk.Widget, conf *Configuration) *homogeneousTab {
	t := &homogeneousTab{}
	t.baseTab = NewBaseTab(parent, "Homogeneous", conf)
	t.pattern = t
	addResetBox := tk.NewHPackLayout(parent)
	addResetBox.AddWidget(t.add)
	addResetBox.AddWidget(t.reset)
	t.AddWidget(addResetBox)
	maxDepthBox := tk.NewHPackLayout(parent)
	maxDepthLabel := tk.NewLabel(parent, "Max depth: ")
	maxDepthBox.AddWidget(maxDepthLabel)
	t.maxDepth = tk.NewEntry(parent)
	t.maxDepth.SetText("6")
	maxDepthBox.AddWidget(t.maxDepth)
	t.AddWidget(maxDepthBox)
	innerPortalsBox := tk.NewHPackLayout(parent)
	innerPortalsLabel := tk.NewLabel(parent, "Inner portals positions: ")
	innerPortalsBox.AddWidget(innerPortalsLabel)
	t.innerPortals = tk.NewComboBox(parent, tk.ComboBoxAttrState(tk.StateReadOnly))
	// The keep together options doesn't seem to work that well, so disable it for now.
	t.innerPortals.SetValues([]string{"Arbitrary", "Spread around (slow)" /*, "Try keep together (slow)"*/})
	t.innerPortals.SetCurrentIndex(0)
	t.innerPortals.OnSelected(func() { t.innerPortals.Entry().ClearSelection() })
	innerPortalsBox.AddWidget(t.innerPortals)
	t.AddWidget(innerPortalsBox)
	strategyBox := tk.NewHPackLayout(parent)
	strategyLabel := tk.NewLabel(parent, "Top triangle: ")
	strategyBox.AddWidget(strategyLabel)
	t.strategy = tk.NewComboBox(parent, tk.ComboBoxAttrState(tk.StateReadOnly))
	t.strategy.SetValues([]string{"Arbitrary", "Largest Area", "Smallest Area", "Most Equilateral", "Random"})
	t.strategy.SetCurrentIndex(0)
	t.strategy.OnSelected(func() { t.strategy.Entry().ClearSelection() })
	strategyBox.AddWidget(t.strategy)
	t.AddWidget(strategyBox)
	t.perfect = tk.NewCheckButton(parent, "Perfect")
	t.AddWidgetEx(t.perfect, tk.FillNone, true, tk.AnchorWest)
	solutionBox := tk.NewHPackLayout(parent)
	solutionBox.AddWidget(t.find)
	solutionBox.AddWidget(t.save)
	solutionBox.AddWidget(t.copy)
	solutionBox.AddWidget(t.solutionLabel)
	t.AddWidget(solutionBox)
	t.AddWidgetEx(t.progress, tk.FillBoth, true, tk.AnchorWest)
	t.AddWidgetEx(t.portalList, tk.FillBoth, true, tk.AnchorWest)

	t.anchorPortals = make(map[string]bool)
	return t
}

func (t *homogeneousTab) onReset() {
	t.anchorPortals = make(map[string]bool)
}

func (t *homogeneousTab) portalLabel(guid string) string {
	if t.disabledPortals[guid] {
		return "Disabled"
	}
	if t.anchorPortals[guid] {
		return "Anchor"
	}
	return "Normal"
}

func (t *homogeneousTab) portalColor(guid string) string {
	if t.disabledPortals[guid] {
		if !t.selectedPortals[guid] {
			return "gray"
		}
		return "dark gray"
	}
	if t.anchorPortals[guid] {
		if !t.selectedPortals[guid] {
			return "green"
		}
		return "dark green"
	}
	if !t.selectedPortals[guid] {
		return "orange"
	}
	return "red"
}

func (t *homogeneousTab) onPortalContextMenu(guid string, x, y int) {
	menu := NewHomogeneousPortalContextMenu(tk.RootWindow(), guid, t)
	tk.PopupMenu(menu.Menu, x, y)
}

func (t *homogeneousTab) search() {
	if len(t.portals) < 3 {
		return
	}
	portals := []lib.Portal{}
	anchors := []int{}
	for _, portal := range t.portals {
		if !t.disabledPortals[portal.Guid] {
			portals = append(portals, portal)
			if t.anchorPortals[portal.Guid] {
				anchors = append(anchors, len(portals)-1)
			}
		}
	}

	options := []lib.HomogeneousOption{}
	maxDepth, err := strconv.Atoi(t.maxDepth.Text())
	if err != nil || maxDepth < 1 {
		return
	}
	options = append(options, lib.HomogeneousMaxDepth(maxDepth))
	options = append(options, lib.HomogeneousPerfect(t.perfect.IsChecked()))
	// set inner portals opion before setting top level scorer, as inner scorer
	// overwrites the top level scorer
	if t.innerPortals.CurrentIndex() == 1 {
		options = append(options, lib.HomogeneousSpreadAround(len(portals)))
		//	} else if t.innerPortals.CurrentIndex() == 2 {
		//		options = append(options, lib.HomogeneousClumpTogether(len(portals)))
	}
	if t.strategy.CurrentIndex() == 1 {
		options = append(options, lib.HomogeneousLargestArea{})
	} else if t.strategy.CurrentIndex() == 2 {
		options = append(options, lib.HomogeneousSmallestArea{})
	} else if t.strategy.CurrentIndex() == 3 {
		options = append(options, lib.HomogeneousMostEquilateralTriangle{})
	} else if t.strategy.CurrentIndex() == 4 {
		rand := rand.New(rand.NewSource(time.Now().UnixNano()))
		options = append(options, lib.HomogeneousRandom{rand})
	}
	options = append(options, lib.HomogeneousProgressFunc(
		func(val int, max int) { t.onProgress(val, max) }))
	t.add.SetState(tk.StateDisable)
	t.reset.SetState(tk.StateDisable)
	t.maxDepth.SetState(tk.StateDisable)
	t.innerPortals.SetState(tk.StateDisable)
	t.perfect.SetState(tk.StateDisable)
	t.strategy.SetState(tk.StateDisable)
	t.find.SetState(tk.StateDisable)
	t.save.SetState(tk.StateDisable)
	t.copy.SetState(tk.StateDisable)
	tk.Update()
	options = append(options, lib.HomogeneousFixedCornerIndices(anchors))

	t.solution, t.depth = lib.DeepestHomogeneous(portals, options...)

	if t.solutionMap != nil {
		t.solutionMap.SetSolution(lib.HomogeneousPolylines(t.depth, t.solution))
	}
	solutionText := fmt.Sprintf("Solution depth: %d", t.depth)
	t.solutionLabel.SetText(solutionText)
	t.add.SetState(tk.StateNormal)
	t.reset.SetState(tk.StateNormal)
	t.maxDepth.SetState(tk.StateNormal)
	t.innerPortals.SetState(tk.StateReadOnly)
	t.perfect.SetState(tk.StateNormal)
	t.strategy.SetState(tk.StateReadOnly)
	t.find.SetState(tk.StateNormal)
	t.save.SetState(tk.StateNormal)
	t.copy.SetState(tk.StateNormal)
	tk.Update()
}

func (t *homogeneousTab) solutionString() string {
	return lib.HomogeneousDrawToolsString(t.depth, t.solution)
}
func (t *homogeneousTab) EnablePortal(guid string) {
	delete(t.disabledPortals, guid)
	t.portalStateChanged(guid)
}
func (t *homogeneousTab) DisablePortal(guid string) {
	t.disabledPortals[guid] = true
	delete(t.anchorPortals, guid)
	t.portalStateChanged(guid)
}
func (t *homogeneousTab) MakeAnchor(guid string) {
	t.anchorPortals[guid] = true
	t.portalStateChanged(guid)
}
func (t *homogeneousTab) UnmakeAnchor(guid string) {
	delete(t.anchorPortals, guid)
	t.portalStateChanged(guid)
}

type homogeneousPortalContextMenu struct {
	*tk.Menu
}

func NewHomogeneousPortalContextMenu(parent tk.Widget, guid string, t *homogeneousTab) *homogeneousPortalContextMenu {
	l := &homogeneousPortalContextMenu{}
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
	if t.anchorPortals[guid] {
		unanchorAction := tk.NewAction("Unmake anchor")
		unanchorAction.OnCommand(func() { t.UnmakeAnchor(guid) })
		l.AddAction(unanchorAction)
	} else if !t.disabledPortals[guid] && len(t.anchorPortals) < 3 {
		anchorAction := tk.NewAction("Make anchor")
		anchorAction.OnCommand(func() { t.MakeAnchor(guid) })
		l.AddAction(anchorAction)
	}
	return l
}
