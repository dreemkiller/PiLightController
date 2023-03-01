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

var lrl_sw = service.NewStatelessProgrammableSwitch()
var fl_sw = service.NewStatelessProgrammableSwitch()

func Setup(lrl_chan *chan int, fl_chan *chan int) {
	fmt.Println("homelink_controller.Setup started")
	// Create the switch accessory.
	a0 := accessory.New(accessory.Info{
		Name: "LivingRoomLightSwitch",
	},
		accessory.TypeProgrammableSwitch)
	a0.AddS(lrl_sw.S)
	a1 := accessory.New(accessory.Info{
		Name: "FoyerLightSwitch",
	},
		accessory.TypeProgrammableSwitch)
	a1.AddS(fl_sw.S)
	// Store the data in the "./db" directory.
	fs := hap.NewFsStore("./db")

	// Create the hap server.
	server, err := hap.NewServer(fs, a0, a1)
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
	go listen_to_chan(fl_chan, fl_sw)
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
