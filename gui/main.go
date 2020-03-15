package main

import "github.com/pwiecz/atk/tk"

func main() {
	conf := LoadConfiguration()
	tk.MainLoop(func() {
		tk.SetMenuTearoff(false)
		mw := NewWindow(conf)
		mw.SetTitle("Portal patterns")
		mw.Center()
		mw.ResizeN(640, 480)
		mw.ShowNormal()
	})
}

type Window struct {
	*tk.Window
	tab *tk.Notebook
}

func NewWindow(conf *Configuration) *Window {
	mw := &Window{}
	mw.Window = tk.RootWindow()
	mw.tab = tk.NewNotebook(mw)

	mw.tab.AddTab(NewHomogeneousTab(mw, conf), "Homogeneous")
	mw.tab.AddTab(NewHerringboneTab(mw, conf), "Herringbone")
	mw.tab.AddTab(NewDoubleHerringboneTab(mw, conf), "Double herringbone")
	mw.tab.AddTab(NewCobwebTab(mw, conf), "Cobweb")

	vbox := tk.NewVPackLayout(mw)
	vbox.AddWidgetEx(mw.tab, tk.FillBoth, true, 0)
	return mw
}
