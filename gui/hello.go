package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/pwiecz/go-fltk"
	"github.com/pwiecz/portal_patterns/configuration"

	guigl "github.com/pwiecz/portal_patterns/gui/gl"
)

import "C"

func main() {
	runtime.LockOSThread()
	conf := configuration.LoadConfiguration()
	w := fltk.NewWindow(800, 600)
	w.Begin()
	t := fltk.NewTabs(10, 10, 780, 580)
	w.Resizable(t)
	NewHomogeneousTab(conf)
	herringbone := fltk.NewPack(20, 30, 760, 550, "Herringbone")
	herringbone.End()

	//	b.Box(fltk.UP_BOX)
	//	b.LabelFont(fltk.HELVETICA_BOLD_ITALIC)
	//	b.LabelSize(36)
	//	b.LabelType(fltk.SHADOW_LABEL)
	t.End()
	w.End()

	fltk.Lock()
	w.Show()
	//	fltk.Run(func() bool { return true })
	fltk.Run()
}

var inited bool = false

func DrawMap(glWin *fltk.GlWindow, window *guigl.MapWindow) {
	if !glWin.Valid() {
		if err := gl.Init(); err != nil {
			log.Fatal("Cannot initialize OpenGL", err)
		}
	}
	if !glWin.ContextValid() {
		fmt.Println("Initializing glWindow")
		window.Init()
	}
	//	if !inited {
	//		window.Init()
	//		inited = true
	//	}

	window.Update()
}
