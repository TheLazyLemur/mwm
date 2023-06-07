package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

func main() {
	xStuff()
}

func xStuff() {
	go func() {
		time.Sleep(time.Second * 10)
		cmd := exec.Command("nitrogen", "--restore")
		_ = cmd.Run()

		cmd = exec.Command("sxhkd", "&")
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
	// go drawWindow(X, screen)

	xproto.CreateWindow(X, screen.RootDepth, root, screen.Root,
		0, 0, 500, 500, 0,
		xproto.WindowClassInputOutput, screen.RootVisual, 0, []uint32{})

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
