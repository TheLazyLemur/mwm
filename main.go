package main

import (
	"log"
	"os/exec"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

func main() {
	X, err := xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}
	defer X.Close()

	screen := xproto.Setup(X).DefaultScreen(X)
	root := screen.Root

	// Create a white window
	window, err := xproto.NewWindowId(X)
	if err != nil {
		log.Fatal(err)
	}

	xproto.CreateWindow(X, screen.RootDepth, window, root,
		0, 0, 500, 500, 0,
		xproto.WindowClassInputOutput, screen.RootVisual, 0, []uint32{})

	xproto.ChangeWindowAttributes(X, window,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			0xffffffff,
			xproto.EventMaskExposure,
		})

	xproto.MapWindow(X, window)

	for {
		event, err := X.WaitForEvent()
		if err != nil {
			log.Fatal(err)
		}

		switch event.(type) {
		case xproto.ExposeEvent:
			// Redraw the window when an Expose event occurs
			drawWindow(X, screen)
		}
	}
}

func drawWindow(X *xgb.Conn, screen *xproto.ScreenInfo) {
	wid, _ := xproto.NewWindowId(X)
	xproto.CreateWindow(X, screen.RootDepth, wid, screen.Root,
		0, 0, 500, 500, 0,
		xproto.WindowClassInputOutput, screen.RootVisual, 0, []uint32{})

	xproto.ChangeWindowAttributes(X, wid,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{ // values must be in the order defined by the protocol
			0xffffffff,
			xproto.EventMaskStructureNotify |
				xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease})

	cmd := exec.Command("nitrogen", "--restore")
	cmd.Run()
}
