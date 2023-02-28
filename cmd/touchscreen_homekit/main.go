package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/dreemkiller/TouchScreenHomekit/homelink_controller"
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

type RoomChannels struct {
}

type Room struct {
	Name      string
	Floor     uint32
	XMin      float64
	XMax      float64
	YMin      float64
	YMax      float64
	ChannelId uint32
}

var floors_pixbuf [2]FloorPixbuf

var current_floor_num FloorNumber

var drawable_floorplan DrawableFloorplanArea

var rooms []Room
var channels = []chan bool{
	make(chan bool),
	make(chan bool),
}

var signals = map[string]interface{}{
	"on_floor_button_clicked":           on_floor_button_clicked,
	"floorplan_button_press_event_cb":   floorplan_button_press_event_cb,
	"floorplan_button_release_event_cb": floorplan_button_release_event_cb,
	"floorplan_touch_event_cb":          floorplan_touch_event_cb,
}

func main() {
	println("Started")
	//lrl_chan := make(chan bool)
	go homelink_controller.Setup(&channels[1])
	// load the room data
	room_data, err := ioutil.ReadFile("rooms.json")
	if err != nil {
		println("Unable to read rooms.json file:", err)
		log.Fatalln("Unable to read rooms.json file:", err)
	}
	err = json.Unmarshal([]byte(room_data), &rooms)
	if err != nil {
		println("json.Unmarshal failed:", err)
		log.Fatalln("json.Unmarshal failed:", err)
	}

	println("Rooms:", rooms)

	const appId = "com.dreemkiller.light_controller"
	app, err := gtk.ApplicationNew(appId, glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		println("Failed to create application")
		log.Fatalln("Couldn't create app:", err)
	}

	current_floor_num.mu.Lock()
	current_floor_num.floor = 0
	current_floor_num.mu.Unlock()

	first_floor_pix, err := gdk.PixbufNewFromFileAtScale("Floorplan_first_1bpp_flipped_cropped.bmp", 427, 320, false)
	if err != nil {
		println("Failed to create pix from file:", err)
		log.Fatalln("Failed to create pix from file:", err)
	}
	floors_pixbuf[0].mu.Lock()
	floors_pixbuf[0].buf = first_floor_pix
	floors_pixbuf[0].mu.Unlock()

	second_floor_pix, err := gdk.PixbufNewFromFileAtScale("Floorplan_second_1bpp_flipped_cropped.bmp", 427, 320, false)
	if err != nil {
		println("Failed to create pix from second floor file:", err)
		log.Fatalln("Failed to create pix from second floor file:", err)
	}
	floors_pixbuf[1].mu.Lock()
	floors_pixbuf[1].buf = second_floor_pix
	floors_pixbuf[1].mu.Unlock()

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
			floors_pixbuf[0].mu.RLock()
			drawable_floorplan_area.SetFromPixbuf(floors_pixbuf[0].buf)
			floors_pixbuf[0].mu.RUnlock()

			drawable_floorplan.mu.Lock()
			drawable_floorplan.image = drawable_floorplan_area
			drawable_floorplan.mu.Unlock()
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

func on_floor_button_clicked(button *gtk.Button) {
	println("Floor button clicked")
	println("button:", button)

	current_floor_num.mu.Lock()
	current_floor_num.floor = (current_floor_num.floor + 1) % 2
	current_floor := current_floor_num.floor
	current_floor_num.mu.Unlock()

	floors_pixbuf[current_floor].mu.RLock()
	var pixbuf = floors_pixbuf[current_floor].buf
	floors_pixbuf[current_floor].mu.RUnlock()

	drawable_floorplan.mu.Lock()
	//drawable_floorplan.image = drawable_floorplan_area
	drawable_floorplan.image.SetFromPixbuf(pixbuf)
	drawable_floorplan.mu.Unlock()
}

var press_start_time time.Time
var long_click_threshold_ms int64 = 250

func floorplan_button_press_event_cb(eventbox *gtk.EventBox, event *gdk.Event) {
	press_start_time = time.Now()
	println("Floorplan button press")
}

func floorplan_button_release_event_cb(eventbox *gtk.EventBox, event *gdk.Event) {
	press_length := time.Now().Sub(press_start_time).Milliseconds()
	fmt.Println("press length:", press_length)
	long_click := false
	if press_length > long_click_threshold_ms {
		long_click = true
	}
	event_button := gdk.EventButtonNewFromEvent(event)
	println("X:", event_button.X())
	println("Y:", event_button.Y())

	x := event_button.X()
	y := event_button.Y()

	current_floor_num.mu.RLock()
	current_floor := current_floor_num.floor
	current_floor_num.mu.RUnlock()

	for _, this_room := range rooms {
		if this_room.Floor == current_floor {
			if this_room.XMin < x &&
				x < this_room.XMax &&
				this_room.YMin < y &&
				y < this_room.YMax {
				println("Click in room ", this_room.Name)
				//if this_room.Responder.Id != 0 {
				// 	go this_room.Responder.TurnOn()
				// 	//this_room.Responder.GetStatus()
				// }
				if this_room.ChannelId != 0 {
					fmt.Println("this_room.ChannelId:", this_room.ChannelId)
					this_channel := channels[this_room.ChannelId]
					fmt.Println("this_room.Chan not nil. Sending down channel")
					if long_click {
						this_channel <- false
					} else {
						this_channel <- true
					}
					fmt.Println("this_room.Chan sent")
				}
				break
			}
		}
	}
}

func floorplan_touch_event_cb(eventbox *gtk.EventBox, event *gdk.Event) {
	println("Floorplan touch event")
	event_button := gdk.EventButtonNewFromEvent(event)
	println("X:", event_button.X())
	println("Y:", event_button.Y())

}
