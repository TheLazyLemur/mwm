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
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const (
	width  = 800
	height = 600
)

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	go xStuff()

	time.Sleep(time.Second * 10)
	// connect to the x server
	// initialize the window manager
	// listen for termination signals to gracefully exit
	// event loop to handle x events
	// Handle the event
	err := glfw.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	window, err := glfw.CreateWindow(width, height, "Drawing Program", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	window.MakeContextCurrent()

	err = gl.Init()
	if err != nil {
		log.Fatal(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version:", version)

	gl.Viewport(0, 0, int32(width), int32(height))
	gl.ClearColor(0.2, 0.3, 0.4, 1.0)

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		drawRectangle()

		window.SwapBuffers()
		glfw.PollEvents()
	}

}

func drawRectangle() {
	gl.Begin(gl.QUADS)
	gl.Color3f(1.0, 0.0, 0.0) // Set color to red
	gl.Vertex2f(-0.5, -0.5)   // Bottom-left vertex
	gl.Vertex2f(0.5, -0.5)    // Bottom-right vertex
	gl.Vertex2f(0.5, 0.5)     // Top-right vertex
	gl.Vertex2f(-0.5, 0.5)    // Top-left vertex
	gl.End()
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
