package main

import (
	"log"
	"os"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var signals = map[string]interface{}{
	"on_floor_button_clicked":         on_floor_button_clicked,
	"floorplan_button_press_event_cb": floorplan_button_press_event_cb,
	"floorplan_touch_event_cb":        floorplan_touch_event_cb,
}

func main() {
	println("Started")
	const appId = "com.dreemkiller.light_controller"
	app, err := gtk.ApplicationNew(appId, glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		println("Failed to create application")
		log.Fatalln("Couldn't create app:", err)
	}

	first_floor_pix, err := gdk.PixbufNewFromFileAtScale("Floorplan_first_1bpp.bmp", 427, 320, true)
	if err != nil {
		println("Failed to create pix from file:", err)
		log.Fatalln("Failed to create pix from file:", err)
	}

	println("Attempting to connect activate")
	app.Connect("activate", func() {
		println("Attempting to load glade file")
		builder, err := gtk.BuilderNewFromFile("LightControllerGUI.glade")
		if err != nil {
			println("Failed to load glade file")
			log.Fatalln("Couldn't make builder:", err)
		}

		builder.ConnectSignals(signals)

		floorplan_area, err := builder.GetObject("Floorplan")
		if err != nil {
			println("Failed to get object Floorplan:", err)
			log.Fatalln("Failed to get objec Floorplan:", err)

		}

		if drawable_floorplan_area, ok := floorplan_area.(*gtk.Image); ok {
			drawable_floorplan_area.SetFromPixbuf(first_floor_pix)
		} else {
			println("Floorplan area is NOT image")
		}

		obj, err := builder.GetObject("Top")
		if err != nil {
			println("Failed to get object Top")
			log.Fatalln("Coultn'd get object Top")
		}

		println("Alls good so far")
		wnd := obj.(*gtk.Window)
		wnd.ShowAll()
		app.AddWindow(wnd)
	})

	app.Run(os.Args)
}

func on_floor_button_clicked() {
	println("Floor button clicked")
}

func floorplan_button_press_event_cb() {
	println("Floorplan pressed")
}

func floorplan_touch_event_cb() {
	println("Floorplan touch event")
}
