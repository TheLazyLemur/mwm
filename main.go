package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"os/exec"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	xStuff()
}

func xStuff() {
	go func() {
		go func() {
			time.Sleep(time.Second * 10)
			cmd := exec.Command("nitrogen", "--restore")
			_ = cmd.Run()
		}()

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
	X.Sync()

	go drawWindow(X, screen)
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

	// This call to ChangeWindowAttributes could be factored out and
	// included with the above CreateWindow call, but it is left here for
	// instructive purposes. It tells X to send us events when the 'structure'
	// of the window is changed (i.e., when it is resized, mapped, unmapped,
	// etc.) and when a key press or a key release has been made when the
	// window has focus.
	// We also set the 'BackPixel' to white so that the window isn't butt ugly.
	xproto.ChangeWindowAttributes(X, wid,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{ // values must be in the order defined by the protocol
			0xffffffff,
			xproto.EventMaskStructureNotify |
				xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease})

	// MapWindow makes the window we've created appear on the screen.
	// We demonstrated the use of a 'checked' request here.
	// A checked request is a fancy way of saying, "do error handling
	// synchronously." Namely, if there is a problem with the MapWindow request,
	// we'll get the error *here*. If we were to do a normal unchecked
	// request (like the above CreateWindow and ChangeWindowAttributes
	// requests), then we would only see the error arrive in the main event
	// loop.
	//
	// Typically, checked requests are useful when you need to make sure they
	// succeed. Since they are synchronous, they incur a round trip cost before
	// the program can continue, but this is only going to be noticeable if
	// you're issuing tons of requests in succession.
	//
	// Note that requests without replies are by default unchecked while
	// requests *with* replies are checked by default.
	err := xproto.MapWindowChecked(X, wid).Check()
	if err != nil {
		fmt.Printf("Checked Error for mapping window %d: %s\n", wid, err)
	} else {
		fmt.Printf("Map window %d successful!\n", wid)
	}

	// This is an example of an invalid MapWindow request and what an error
	// looks like.
	err = xproto.MapWindowChecked(X, 0).Check()
	if err != nil {
		fmt.Printf("Checked Error for mapping window 0x1: %s\n", err)
	} else { // neva
		fmt.Printf("Map window 0x1 successful!\n")
	}

	// Start the main event loop.
	for {
		// WaitForEvent either returns an event or an error and never both.
		// If both are nil, then something went wrong and the loop should be
		// halted.
		//
		// An error can only be seen here as a response to an unchecked
		// request.
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
