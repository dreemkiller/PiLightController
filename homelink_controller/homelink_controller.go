package homelink_controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/service"
)

type Room struct {
	RoomName   string
	Floor      uint32
	XMin       float64
	XMax       float64
	YMin       float64
	YMax       float64
	ChannelId  int32
	SwitchName string
}

func Setup(channels *[]chan int, rooms *[]Room, max_channel_id int32) {
	fmt.Println("homelink_controller.Setup started")
	accessories := make([]*accessory.A, max_channel_id+1)
	switches := make([]service.StatelessProgrammableSwitch, max_channel_id+1)
	for i := int32(0); i <= max_channel_id; i++ {
		// Figure out which room has this channel ID
		var this_room Room
		found := false
		for _, this_room = range *rooms {
			if this_room.ChannelId == i {
				found = true
				break
			}
		}
		if !found {
			log.Panic("We should not be here")
		}
		// Create the switch accessory.
		accessories[i] = accessory.New(
			accessory.Info{
				Name: this_room.SwitchName,
			},
			accessory.TypeProgrammableSwitch,
		)
		// Create the switch
		switches[i] = *service.NewStatelessProgrammableSwitch()
		// Add the switch to the accessory
		accessories[i].AddS(switches[i].S)
	}
	// Store the data in the "./db" directory.
	fs := hap.NewFsStore("./db")

	// Create the hap server.
	server, err := hap.NewServer(fs, accessories[0], accessories[1:]...)
	if err != nil {
		// stop if an error happens
		log.Panic(err)
	}

	// Setup a listener for interrupts and SIGTERM signals
	// to stop the server.
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		// Stop delivering signals.
		signal.Stop(c)
		// Cancel the context to stop the server.
		cancel()
	}()

	for i := int32(0); i <= max_channel_id; i++ {
		go listen_to_chan(&(*channels)[i], &switches[i])
	}
	// Run the server.
	fmt.Println("homelink_controller.Setup Listening")
	server.ListenAndServe(ctx)
}

func listen_to_chan(lrl_chan *chan int, sw *service.StatelessProgrammableSwitch) {
	for {
		fmt.Println("listen_to_chan started")
		new_value := <-(*lrl_chan)
		fmt.Println("listen_to_chan passed the channel")
		sw.ProgrammableSwitchEvent.Int.SetValue(new_value)
		fmt.Println("listen_to_chan completed")
	}
}
