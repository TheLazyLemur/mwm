package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

func main() {
	xStuff()
}

func xStuff() {
	go func() {
		// go func() {
		// 	time.Sleep(time.Second * 10)
		// 	cmd := exec.Command("nitrogen", "--restore")
		// 	_ = cmd.Run()
		// }()

		X, err := xgb.NewConn()
		if err != nil {
			log.Fatal(err)
		}
		defer X.Close()

		err = initialize(X)
		if err != nil {
			log.Fatal(err)
		}

		setupSignalHandler()

		for {
			event, err := X.WaitForEvent()
			if err != nil {
				log.Fatal(err)
			}

			switch event := event.(type) {
			case xproto.MapRequestEvent:
				handleMapRequest(X, event)
			case xproto.ConfigureRequestEvent:
				handleConfigureRequest(X, event)
			}
		}
	}()
}

func initialize(X *xgb.Conn) error {
	// Get the root window
	screen := xproto.Setup(X).DefaultScreen(X)
	root := screen.Root

	// Select events we are interested in
	err := xproto.ChangeWindowAttributesChecked(X, root, xproto.CwEventMask, []uint32{xproto.EventMaskSubstructureRedirect}).Check()
	if err != nil {
		return fmt.Errorf("unable to change window attributes: %v", err)
	}

	// Flush the request to the X server
	go drawWindow(X, screen)
	X.Sync()

	return nil
}

func handleMapRequest(X *xgb.Conn, event xproto.MapRequestEvent) {
	// Create the window
	window := event.Window

	// Configure the window
	err := xproto.ConfigureWindowChecked(X, window, xproto.ConfigWindowStackMode, []uint32{xproto.StackModeAbove}).Check()
	if err != nil {
		log.Printf("unable to configure window: %v", err)
	}

	// Map the window
	err = xproto.MapWindowChecked(X, window).Check()
	if err != nil {
		log.Printf("unable to map window: %v", err)
	}
}

func handleConfigureRequest(X *xgb.Conn, event xproto.ConfigureRequestEvent) {
	// Configure the window according to the request
	values := []uint32{
		uint32(event.X),
		uint32(event.Y),
		uint32(event.Width),
		uint32(event.Height),
		uint32(event.Sibling),
		uint32(event.StackMode),
	}
	err := xproto.ConfigureWindowChecked(X, event.Window, xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight|xproto.ConfigWindowSibling|xproto.ConfigWindowStackMode, values).Check()
	if err != nil {
		log.Printf("unable to configure window: %v", err)
	}
}

func setupSignalHandler() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for sig := range signalCh {
			log.Printf("Received signal: %v. Exiting...", sig)
			os.Exit(0)
		}
	}()
}

func drawWindow(X *xgb.Conn, screen *xproto.ScreenInfo) {
	// Any time a new resource (i.e., a window, pixmap, graphics context, etc.)
	// is created, we need to generate a resource identifier.
	// If the resource is a window, then use xproto.NewWindowId. If it's for
	// a pixmap, then use xproto.NewPixmapId. And so on...
	wid, _ := xproto.NewWindowId(X)

	// CreateWindow takes a boatload of parameters.
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

	err := xproto.MapWindowChecked(X, wid).Check()
	if err != nil {
		fmt.Printf("Checked Error for mapping window %d: %s\n", wid, err)
	} else {
		fmt.Printf("Map window %d successful!\n", wid)
	}

	err = xproto.MapWindowChecked(X, 0).Check()
	if err != nil {
		fmt.Printf("Checked Error for mapping window 0x1: %s\n", err)
	} else { // neva
		fmt.Printf("Map window 0x1 successful!\n")
	}

	for {
		ev, xerr := X.WaitForEvent()
		if ev == nil && xerr == nil {
			fmt.Println("Both event and error are nil. Exiting...")
			return
		}

		if ev != nil {
			fmt.Printf("Event: %s\n", ev)
		}
		if xerr != nil {
			fmt.Printf("Error: %s\n", xerr)
		}
	}
}
