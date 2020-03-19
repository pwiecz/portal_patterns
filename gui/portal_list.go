package main

import "sort"

import "github.com/pwiecz/atk/tk"
import "github.com/pwiecz/portal_patterns/lib"

type PortalList struct {
	*tk.TreeViewEx
	// portal guid to item
	items map[string]*tk.TreeItem
	// item id to portal guid
	guids              map[string]string
	onPortalRightClick func(string, int, int)
}

func NewPortalList(parent tk.Widget) *PortalList {
	l := &PortalList{}
	l.TreeViewEx = tk.NewTreeViewEx(parent)
	l.items = make(map[string]*tk.TreeItem)
	l.guids = make(map[string]string)
	l.SetColumnCount(2)
	l.SetHeaderLabels([]string{"Title", "State"})
	l.BindEvent("<Button-3>", func(e *tk.Event) {
		clickedItem := l.ItemAt(e.PosX, e.PosY)
		l.SetSelections(clickedItem)
		if guid, ok := l.guids[clickedItem.Id()]; ok {
			if l.onPortalRightClick != nil {
				l.onPortalRightClick(guid, e.GlobalPosX, e.GlobalPosY)
			}
		}
	})
	return l
}

func (l *PortalList) Clear() {
	l.DeleteAllItems()
	l.items = make(map[string]*tk.TreeItem)
	l.guids = make(map[string]string)
}

func (l *PortalList) ScrollToPortal(guid string) {
	if item, ok := l.items[guid]; ok {
		l.ScrollTo(item)
	}
}
func (l *PortalList) SelectedPortals() []string {
	items := l.SelectionList()
	var portals []string
	for _, item := range items {
		if guid, ok := l.guids[item.Id()]; ok {
			portals = append(portals, guid)
		}
	}
	return portals
}
func (l *PortalList) SetSelectedPortals(guids map[string]bool) {
	var selectedItems []*tk.TreeItem
	for guid := range guids {
		if item, ok := l.items[guid]; ok {
			selectedItems = append(selectedItems, item)
		}
	}
	l.SetSelectionList(selectedItems)
}

func (l *PortalList) OnPortalRightClick(onPortalRightClick func(string, int, int)) {
	l.onPortalRightClick = onPortalRightClick
}

func (l *PortalList) SetPortals(portals map[string]lib.Portal) {
	portalList := []lib.Portal{}
	for _, portal := range portals {
		portalList = append(portalList, portal)
	}
	sort.Slice(portalList, func(i, j int) bool {
		return portalList[i].Name < portalList[j].Name
	})

	l.Clear()
	for i, portal := range portalList {
		item := l.InsertItem(nil, i, portal.Name, []string{""})
		l.items[portal.Guid] = item
		l.guids[item.Id()] = portal.Guid
	}
}

func (l *PortalList) SetPortalState(guid, state string) {
	if item, ok := l.items[guid]; ok {
		item.SetColumnText(1, state)
	}
}
