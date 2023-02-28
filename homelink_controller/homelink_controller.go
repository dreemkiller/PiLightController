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
)

var lrl_sw = accessory.NewSwitch(accessory.Info{
	Name: "LivingRoomLightSwitch",
})

func Setup(lrl_chan *chan bool) {
	fmt.Println("homelink_controller.Setup started")
	// Create the switch accessory.
	// lrl_sw := accessory.NewSwitch(accessory.Info{
	// 	Name: "LivingRoomLightSwitch",
	// })
	lrl_sw.Switch.On.SetValue(false)

	// Store the data in the "./db" directory.
	fs := hap.NewFsStore("./db")

	// Create the hap server.
	server, err := hap.NewServer(fs, lrl_sw.A)
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

	go listen_to_chan(lrl_chan, lrl_sw)
	// Run the server.
	fmt.Println("homelink_controller.Setup Listening")
	server.ListenAndServe(ctx)
}

func listen_to_chan(lrl_chan *chan bool, sw *accessory.Switch) {
	for {
		fmt.Println("listen_to_chan started")
		new_value := <-(*lrl_chan)
		fmt.Println("listen_to_chan passed the channel")
		sw.Switch.On.SetValue(new_value)
		fmt.Println("listen_to_chan completed")
	}
}
