package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/pwiecz/portal_patterns/lib"
)

type PortalList struct {
	widget.Table

	// portal guid to item
	portals             []lib.Portal
	OnPortalSelected    func(string, bool)
	OnPortalContextMenu func(string, float32, float32)
	OnContextMenu       func(float32, float32)
	portalLabel         func(string) string
}

var _ desktop.Mouseable = (*PortalList)(nil)

func NewPortalList(portalLabel func(string) string) *PortalList {
	l := &PortalList{}
	l.portalLabel = portalLabel
	l.Length = l.tableSize
	l.CreateCell = l.tableCreate
	l.UpdateCell = l.tableUpdate
	l.ExtendBaseWidget(l)
	return l
}

func (l *PortalList) tableSize() (int, int) {
	return len(l.portals), 2
}
func (t *PortalList) tableCreate() fyne.CanvasObject {
	return widget.NewLabel("                    ")
}
func (t *PortalList) tableUpdate(id widget.TableCellID, canvasObject fyne.CanvasObject) {
	if label, ok := canvasObject.(*widget.Label); ok {
		if id.Col == 0 {
			label.SetText(t.portals[id.Row].Name)
		} else {
			label.SetText(t.portalLabel(t.portals[id.Row].Guid))
		}
	}
}
func (l *PortalList) onTableCellSelected(id widget.TableCellID) {
	fmt.Println("Selected", id.Row, id.Col)
}
func (l *PortalList) onTableCellUnselected(id widget.TableCellID) {
	fmt.Println("Unselected", id.Row, id.Col)
}
func (l *PortalList) Clear() {
	l.portals = nil
	l.Refresh()
}

func (l *PortalList) SetPortals(portals []lib.Portal) {
	l.portals = nil
	l.portals = append(l.portals, portals...)
	l.Refresh()
}

func (l *PortalList) MouseDown(event *desktop.MouseEvent) {
}
func (l *PortalList) MouseUp(event *desktop.MouseEvent) {
	fmt.Println("mouseup")
	if event.Button == desktop.MouseButtonSecondary {
		if l.OnContextMenu != nil {
			fmt.Println("oncontextmenu", event.Position.X, event.Position.Y)
			l.OnContextMenu(event.Position.X, event.Position.Y)
		}
	}
}

/*func (l *PortalList) ScrollToPortal(guid string) {
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

func (l *PortalList) SetPortals(portals []lib.Portal) {
	// local copy not to modify caller's the slice
	portalList := append(([]lib.Portal)(nil), portals...)
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
*/
