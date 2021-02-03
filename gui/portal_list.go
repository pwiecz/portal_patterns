package main

import (
	"fmt"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/lib"
)

type portalList struct {
	*fltk.TableRow
	portals                 []lib.Portal
	selectedPortals         map[string]struct{}
	portalState             map[string]string
	selectionChangeCallback func()
	contextMenuCallback     func(int, int)
}

func newPortalList(x, y, w, h int) *portalList {
	l := &portalList{
		selectedPortals: make(map[string]struct{}),
	}
	l.TableRow = fltk.NewTableRow(0, 0, 100, 540)

	l.SetColumnCount(2)
	l.SetDrawCellCallback(func(context fltk.TableContext, r, c, x, y, w, h int) {
		l.drawCallback(context, r, c, x, y, w, h)
	})
	l.EnableColumnHeaders()
	l.AllowColumnResizing()
	l.SetColumnWidth(0, 200)
	l.SetEventHandler(func(event fltk.Event) bool {
		return l.onEvent(event)
	})
	l.SetCallbackCondition(fltk.WhenRelease)
	l.SetCallback(func() {
		l.onRelease()
	})
	l.SetType(fltk.SelectMulti)

	return l
}

func (l *portalList) SetSelectionChangeCallback(callback func()) {
	l.selectionChangeCallback = callback
}
func (l *portalList) SetContextMenuCallback(callback func(int, int)) {
	l.contextMenuCallback = callback
}
func (l *portalList) SetPortals(portals []lib.Portal) {
	l.portals = portals
	l.SetRowCount(len(portals))
}

func (l *portalList) drawCallback(context fltk.TableContext, row, column, x, y, w, h int) {
	switch context {
	case fltk.ContextCell:
		if row >= len(l.portals) {
			return
		}
		background := uint(0xffffffff)
		if l.IsRowSelected(row) {
			background = l.SelectionColor()
		}
		fltk.DrawBox(fltk.THIN_UP_BOX, x, y, w, h, background)
		fltk.Color(0x00000000)
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
		fltk.Color(0x00000000)
		if column == 0 {
			fltk.Draw("Name", x, y, w, h, fltk.ALIGN_CENTER)
		} else if column == 1 {
			fltk.Draw("State", x, y, w, h, fltk.ALIGN_CENTER)
		}
	}
}

func (l *portalList) onEvent(event fltk.Event) bool {
	if event == fltk.RELEASE {
		l.onSelectionMaybeChanged()
	}
	return false
}

func (l *portalList) onRelease() {
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

func (l *portalList) onSelectionMaybeChanged() {
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
				fmt.Println("selected:", l.portals[i].Name)
				l.selectedPortals[l.portals[i].Guid] = struct{}{}
			}
		}
		if l.selectionChangeCallback != nil {
			l.selectionChangeCallback()
		}
	}
}
