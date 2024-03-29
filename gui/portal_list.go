package main

import (
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/lib"
)

type PortalList struct {
	*fltk.TableRow
	portals                 []lib.Portal
	portalIndices           map[string]int
	selectedPortals         map[string]struct{}
	portalState             map[string]string
	selectionChangeCallback func()
	contextMenuCallback     func(int, int)
}

func NewPortalList(x, y, w, h int) *PortalList {
	l := &PortalList{
		selectedPortals: make(map[string]struct{}),
		portalState:     make(map[string]string),
	}
	l.TableRow = fltk.NewTableRow(x, y, w, h)

	l.SetColumnCount(2)
	l.SetDrawCellCallback(l.drawCallback)
	l.EnableColumnHeaders()
	scrollSize := fltk.ScrollbarSize()
	l.SetColumnWidth(0, w*5/7-scrollSize/2-1)
	l.SetColumnWidth(1, w*2/7-scrollSize/2-1)
	l.SetEventHandler(l.onEvent)
	l.SetCallbackCondition(fltk.WhenRelease)
	l.SetCallback(l.onRelease)
	l.SetType(fltk.SelectMulti)
	return l
}

func (l *PortalList) SetSelectedPortals(selection map[string]struct{}) {
	l.SelectAllRows(fltk.Deselect)
	for guid := range selection {
		l.SelectRow(l.portalIndices[guid], fltk.Select)
	}
}
func (l *PortalList) SetSelectionChangeCallback(callback func()) {
	l.selectionChangeCallback = callback
}
func (l *PortalList) SetContextMenuCallback(callback func(int, int)) {
	l.contextMenuCallback = callback
}
func (l *PortalList) SetPortals(portals []lib.Portal) {
	l.portals = portals
	l.portalIndices = make(map[string]int)
	for i, portal := range l.portals {
		l.portalIndices[portal.Guid] = i
	}
	l.portalState = make(map[string]string)
	l.SetRowCount(len(portals))
}
func (l *PortalList) SetPortalLabel(guid, label string) {
	l.portalState[guid] = label
}
func (l *PortalList) ScrollToPortal(guid string) {
	portalIndex, ok := l.portalIndices[guid]
	if !ok {
		return
	}
	top, _, bottom, _ := l.VisibleCells()
	if portalIndex >= top && portalIndex <= bottom {
		return
	}
	topRow := portalIndex - (bottom-top)/2
	if topRow < 0 {
		topRow = 0
	}
	l.SetTopRow(topRow)
}

func (l *PortalList) drawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
	switch context {
	case fltk.ContextCell:
		if row >= len(l.portals) {
			return
		}
		background := fltk.WHITE
		if l.IsRowSelected(row) {
			background = l.SelectionColor()
		}
		fltk.DrawBox(fltk.THIN_UP_BOX, x, y, w, h, background)
		fltk.SetDrawColor(fltk.BLACK)
		if column == 0 {
			fltk.Draw(l.portals[row].Name, x, y, w, h, fltk.ALIGN_LEFT)
		} else if column == 1 {
			stateText, ok := l.portalState[l.portals[row].Guid]
			if !ok {
				stateText = "Normal"
			}
			fltk.Draw(stateText, x, y, w, h, fltk.ALIGN_CENTER)
		}
	case fltk.ContextColHeader:
		fltk.DrawBox(fltk.UP_BOX, x, y, w, h, 0x8f8f8fff)
		fltk.SetDrawColor(fltk.BLACK)
		if column == 0 {
			fltk.Draw("Name", x, y, w, h, fltk.ALIGN_CENTER)
		} else if column == 1 {
			fltk.Draw("State", x, y, w, h, fltk.ALIGN_CENTER)
		}
	}
}

func (l *PortalList) onEvent(event fltk.Event) bool {
	if event == fltk.RELEASE {
		l.onSelectionMaybeChanged()
	}
	return false
}

func (l *PortalList) onRelease() {
	if l.CallbackContext() != fltk.ContextCell {
		return
	}
	l.onSelectionMaybeChanged()
	if fltk.EventButton() != fltk.RightMouse || fltk.EventState() != 0 {
		return
	}
	row := l.CallbackRow()
	if !l.IsRowSelected(row) {
		l.SelectAllRows(fltk.Deselect)
		l.SelectRow(row, fltk.Select)
	}
	l.onSelectionMaybeChanged()
	if l.contextMenuCallback != nil {
		l.contextMenuCallback(fltk.EventX(), fltk.EventY())
	}
}

func (l *PortalList) onSelectionMaybeChanged() {
	selectionChanged := false
	numSelectedRows := 0
	for i := 0; i < len(l.portals); i++ {
		if l.IsRowSelected(i) {
			numSelectedRows++
			guid := l.portals[i].Guid
			if _, ok := l.selectedPortals[guid]; !ok {
				selectionChanged = true
				break
			}
		}
	}
	if numSelectedRows != len(l.selectedPortals) {
		selectionChanged = true
	}
	if selectionChanged {
		for key := range l.selectedPortals {
			delete(l.selectedPortals, key)
		}
		for i := 0; i < len(l.portals); i++ {
			if l.IsRowSelected(i) {
				l.selectedPortals[l.portals[i].Guid] = struct{}{}
			}
		}
		if l.selectionChangeCallback != nil {
			l.selectionChangeCallback()
		}
	}
}
