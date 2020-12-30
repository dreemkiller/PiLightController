package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

type FloorPixbuf struct {
	mu  sync.RWMutex
	buf *gdk.Pixbuf
}

type FloorNumber struct {
	mu    sync.RWMutex
	floor uint32
}

type DrawableFloorplanArea struct {
	mu    sync.RWMutex
	image *gtk.Image
}

type Room struct {
	Name      string
	Floor     uint32
	XMin      float64
	XMax      float64
	YMin      float64
	YMax      float64
	Responder InsteonResponder
}

type Rooms struct {
	mu    sync.RWMutex
	array []Room
}

type FloorplanTouchTimeType struct {
	mu   sync.Mutex
	time uint32
}

var floorPixbuf [2]FloorPixbuf

var floorplanTouchTime FloorplanTouchTimeType

var currentFloorNum FloorNumber

var drawableFloorplan DrawableFloorplanArea

var rooms Rooms

var signals = map[string]interface{}{
	"floor_button_pressed_cb":           floor_button_pressed_cb,
	"floorplan_button_press_event_cb":   floorplan_button_press_event_cb,
	"floorplan_button_release_event_cb": floorplan_button_release_event_cb,
}

func main() {
	println("PiLightController started")
	// load the room data
	roomData, err := ioutil.ReadFile("rooms.json")
	if err != nil {
		println("Unable to read rooms.json file:", err)
		log.Fatalln("Unable to read rooms.json file:", err)
	}
	rooms.mu.Lock()
	err = json.Unmarshal([]byte(roomData), &rooms.array)
	if err != nil {
		println("json.Unmarshal failed:", err)
		log.Fatalln("json.Unmarshal failed:", err)
	}
	rooms.mu.Unlock()

	const appID = "com.dreemkiller.light_controller"
	app, err := gtk.ApplicationNew(appID, glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		println("Failed to create application")
		log.Fatalln("Couldn't create app:", err)
	}

	currentFloorNum.mu.Lock()
	currentFloorNum.floor = 0
	currentFloorNum.mu.Unlock()

	firstFloorPix, err := gdk.PixbufNewFromFileAtScale("Floorplan_first_1bpp_flipped_cropped.bmp", 427, 320, false)
	if err != nil {
		println("Failed to create pix from file:", err)
		log.Fatalln("Failed to create pix from file:", err)
	}
	floorPixbuf[0].mu.Lock()
	floorPixbuf[0].buf = firstFloorPix
	floorPixbuf[0].mu.Unlock()

	secondFloorPix, err := gdk.PixbufNewFromFileAtScale("Floorplan_second_1bpp_flipped_cropped.bmp", 427, 320, false)
	if err != nil {
		println("Failed to create pix from second floor file:", err)
		log.Fatalln("Failed to create pix from second floor file:", err)
	}
	floorPixbuf[1].mu.Lock()
	floorPixbuf[1].buf = secondFloorPix
	floorPixbuf[1].mu.Unlock()

	app.Connect("activate", func() {
		builder, err := gtk.BuilderNewFromFile("LightControllerGUI.glade")
		if err != nil {
			println("Failed to load glade file")
			log.Fatalln("Couldn't make builder:", err)
		}

		builder.ConnectSignals(signals)

		floorplanArea, err := builder.GetObject("Floorplan")
		if err != nil {
			println("Failed to get object Floorplan:", err)
			log.Fatalln("Failed to get objec Floorplan:", err)

		}

		if drawableFloorplanArea, ok := floorplanArea.(*gtk.Image); ok {
			floorPixbuf[0].mu.RLock()
			drawableFloorplanArea.SetFromPixbuf(floorPixbuf[0].buf)
			floorPixbuf[0].mu.RUnlock()

			drawableFloorplan.mu.Lock()
			drawableFloorplan.image = drawableFloorplanArea
			drawableFloorplan.mu.Unlock()
		} else {
			println("Floorplan area is NOT image")
			log.Fatalln("Floorplan area is NOT image")
		}

		obj, err := builder.GetObject("Top")
		if err != nil {
			println("Failed to get object Top")
			log.Fatalln("Coultn'd get object Top")
		}

		wnd := obj.(*gtk.Window)
		wnd.ShowAll()
		app.AddWindow(wnd)
	})

	app.Run(os.Args)
}

func floor_button_pressed_cb(button *gtk.Button) {
	println("Floor button clicked")
	println("button:", button)

	currentFloorNum.mu.Lock()
	currentFloorNum.floor = (currentFloorNum.floor + 1) % 2
	currentFloor := currentFloorNum.floor
	currentFloorNum.mu.Unlock()

	floorPixbuf[currentFloor].mu.RLock()
	var pixbuf = floorPixbuf[currentFloor].buf
	floorPixbuf[currentFloor].mu.RUnlock()

	drawableFloorplan.mu.Lock()
	//drawableFloorplan.image = drawableFloorplanArea
	drawableFloorplan.image.SetFromPixbuf(pixbuf)
	drawableFloorplan.mu.Unlock()
}

type EventType int

const (
	EventTypeOn = iota
	EventTypeOff
)

const pressTimeThreshold = 300

func eventType(time uint32) EventType {
	if time > pressTimeThreshold {
		return EventTypeOff
	} else {
		return EventTypeOn
	}
}

func floorplan_button_press_event_cb(eventbox *gtk.EventBox, event *gdk.Event) {
	println("Floorplan button press")
	eventButton := gdk.EventButtonNewFromEvent(event)
	floorplanTouchTime.mu.Lock()
	floorplanTouchTime.time = eventButton.Time()
	floorplanTouchTime.mu.Unlock()
}

func handle_floorplan_event(eventButton *gdk.EventButton) {
	println("handle_floorplan_event started")
	println("X:", eventButton.X())
	println("Y:", eventButton.Y())

	floorplanTouchTime.mu.Lock()
	var startTime = floorplanTouchTime.time
	floorplanTouchTime.mu.Unlock()
	elapsedTime := eventButton.Time() - startTime
	println("Elapsed Time:", elapsedTime)
	var eventType = eventType(elapsedTime)

	x := eventButton.X()
	y := eventButton.Y()
	println("event type:", eventType)

	currentFloorNum.mu.RLock()
	currentFloor := currentFloorNum.floor
	currentFloorNum.mu.RUnlock()

	rooms.mu.RLock()
	for _, thisRoom := range rooms.array {
		if thisRoom.Floor == currentFloor {
			if thisRoom.XMin < x &&
				x < thisRoom.XMax &&
				thisRoom.YMin < y &&
				y < thisRoom.YMax {
				println("In room ", thisRoom.Name)
				if thisRoom.Responder.ID != 0 {
					if eventType == EventTypeOn {
						thisRoom.Responder.TurnOn()
					} else {
						thisRoom.Responder.TurnOff()
					}
				}
				break
			}
		}
	}
	rooms.mu.RUnlock()
}
func floorplan_button_release_event_cb(eventbox *gtk.EventBox, event *gdk.Event) {
	println("Floorplan button release")
	eventButton := gdk.EventButtonNewFromEvent(event)

	// do this in a separate thread to prevent network delays from slowing down the GUI
	go handle_floorplan_event(eventButton)
}
