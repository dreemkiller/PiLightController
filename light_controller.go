package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

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

type Color struct {
	Red uint8
	Green uint8
	Blue uint8
}

var yellow = Color {
	235,
	235,
	52,
}

var floorPixbuf [2]FloorPixbuf

var floorplanTouchTime FloorplanTouchTimeType

var currentFloorNum FloorNumber

var drawableFloorplan DrawableFloorplanArea

var rooms Rooms

var signals = map[string]interface{}{
	"floor_button_pressed_cb":         floor_button_pressed_cb,
	"floorplan_button_press_event_cb": floorplan_button_press_event_cb,
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
	println("firstFloorPix.GetBitsPerSample:", firstFloorPix.GetBitsPerSample())
	pixels := firstFloorPix.GetPixels()
	println("firstFloorPix.GetWidth:", firstFloorPix.GetWidth())
	println("firstFloorPix.GetHeight:", firstFloorPix.GetHeight())
	println("firstFloorPix.GetPixels.len:", len(pixels))
	println("firstFloorPix.GetNChannels():", firstFloorPix.GetNChannels())
	println("firstFloorPix.GetRowStride():", firstFloorPix.GetRowstride())
	println("rowstride * Height:", firstFloorPix.GetRowstride() * firstFloorPix.GetHeight())
	//expected_len := firstFloorPix.GetNChannels() * (firstFloorPix.GetBitsPerSample()/8) * firstFloorPix.GetWidth() * firstFloorPix.GetHeight()
	expected_len := firstFloorPix.GetRowstride() * firstFloorPix.GetHeight()
	println("firstFloorPix.Expected Pixel Length:", expected_len)
	println("difference:", len(pixels) - expected_len)
	println("Colorspace:", firstFloorPix.GetColorspace())

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

		// start the thread to periodically check the status of the devices
		go light_status_loop()

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
	drawableFloorplan.image.SetFromPixbuf(pixbuf)
	drawableFloorplan.mu.Unlock()
}

func floorplan_button_press_event_cb(eventbox *gtk.EventBox, event *gdk.Event) {
	// do this in a separate thread to prevent network delays from slowing down the GUI
	// response time
	go handle_floorplan_event(event)
}

func handle_floorplan_event(event *gdk.Event) {
	eventButton := gdk.EventButtonNewFromEvent(event)
	println("handle_floorplan_event started")

	x := eventButton.X()
	y := eventButton.Y()
	println("X:", x)
	println("Y:", y)

	currentFloorNum.mu.RLock()
	currentFloor := currentFloorNum.floor
	currentFloorNum.mu.RUnlock()

	rooms.mu.RLock()
	for i, thisRoom := range rooms.array {
		if thisRoom.Floor == currentFloor {
			if thisRoom.XMin < x &&
				x < thisRoom.XMax &&
				thisRoom.YMin < y &&
				y < thisRoom.YMax {
				println("In room ", thisRoom.Name)
				if thisRoom.Responder.ID != 0 {
					// not using thisRoom below because we want to modify
					// the contents of the variable, not a copy of it
					toggleRoomColor(thisRoom)
					rooms.array[i].Responder.Toggle()
					
				}
				break
			}
		}
	}
	rooms.mu.RUnlock()
}

func toggleRoomColor(room Room) {
	println("toggleRoom called")
	floorPixbuf[room.Floor].mu.Lock()
	var pixbuf = floorPixbuf[room.Floor].buf
	pixels := pixbuf.GetPixels()
	for y := int(room.YMin); y < int(room.YMax); y++ {
		for x := int(room.XMin); x < int(room.XMax); x++ {
			pixel_byte_offset := (pixbuf.GetRowstride() * y) + (pixbuf.GetBitsPerSample()/8 * pixbuf.GetNChannels() * x)
			pixels[pixel_byte_offset] ^= yellow.Red ^ 0xff
			pixels[pixel_byte_offset + 1] ^= yellow.Green ^ 0xff
			pixels[pixel_byte_offset + 2] ^= yellow.Blue ^ 0xff
		}
	}
	floorPixbuf[room.Floor].mu.Unlock()

	
	currentFloorNum.mu.RLock()
	var currentFloor = currentFloorNum.floor
	currentFloorNum.mu.RUnlock()
	if currentFloor == room.Floor {
		drawableFloorplan.mu.Lock()
		drawableFloorplan.image.SetFromPixbuf(pixbuf)
		drawableFloorplan.mu.Unlock()
	}
}

func light_status_loop() {
	for {
		time.Sleep(5 * time.Second)
		rooms.mu.RLock()
		for i, thisRoom := range rooms.array {
			if thisRoom.Responder.ID != 0 {
				// Not using thisRoom because we want to modify
				// the contents of the variable, not a copy of it
				changed := rooms.array[i].Responder.UpdateStatus()
				if changed {
					if rooms.array[i].Responder.is_on {
						turn_on_room(&rooms.array[i])
					} else {
						turn_off_room(&rooms.array[i])
					}
					toggleRoomColor(rooms.array[i])
				}
			}
		}

	}
}

func turn_on_room(room *Room) {
	println("turn_on_room")
	floorPixbuf[room.Floor].mu.Lock()
	pixels := floorPixbuf[room.Floor].buf.GetPixels()
	println("pixels (length:", len(pixels), "):")
	// for _, byte := range pixels {
	// 	print(byte)
	// }
	println("")
	floorPixbuf[room.Floor].mu.Unlock()
}

func turn_off_room(room *Room) {
	println("turn_off_room")
	floorPixbuf[room.Floor].mu.Lock()
	pixels := floorPixbuf[room.Floor].buf.GetPixels()
	println("pixels (length:", len(pixels), "):")
	// for _, byte := range pixels {
	// 	print(byte)
	// }
	println("")
	floorPixbuf[room.Floor].mu.Unlock()
}
