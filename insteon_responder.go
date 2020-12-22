package main

import (
	"fmt"
	"net/http"
)

const hub_ip = "192.168.0.100"
const hub_port = 25105

const (
	InsteonResponderTypeGroup = iota
	InsteonResponderTypeDevice
)

func NewInsteonCommandPassThrough(id uint32, cmd1 uint8, cmd2 uint8) InsteonCommand {
	return InsteonCommand{
		PassThrough: 262,
		Id:          id,
		Flags:       0xF,
		Cmd1:        cmd1,
		Cmd2:        cmd2,
	}
}

func (ic InsteonCommand) Format() string {
	buf := fmt.Sprintf("%04d", ic.PassThrough)
	buf = fmt.Sprintf("%s%06X", buf, ic.Id)
	buf = fmt.Sprintf("%s%02X", buf, ic.Flags)
	buf = fmt.Sprintf("%s%02X", buf, ic.Cmd1)
	buf = fmt.Sprintf("%s%02X", buf, ic.Cmd2)
	return buf
}

type InsteonResponderType int

type InsteonCommand struct {
	PassThrough uint16 // 0262 for Pass-through to PLM
	Id          uint32 // ID for the command
	Flags       uint8  // FLags Byte (constant)
	Cmd1        uint8  // CMD1
	Cmd2        uint8  // CMD2
}

type InsteonResponder struct {
	Id   uint32
	Type InsteonResponderType
}

func (ir *InsteonResponder) sendCommand(command uint8) {
	host := fmt.Sprintf("http://%s:%d", hub_ip, hub_port)
	var path string
	switch ir.Type {
	case InsteonResponderTypeGroup:
		path = fmt.Sprintf("/3?0262%06x0F%02xFF=I=3\n", ir.Id, command)
	case InsteonResponderTypeDevice:
		//url = fmt.Sprintf("http://%s/0?%x%02d=I=0 HTTP/1.1\nAuthorization: Basic Q2xpZnRvbjg6MEJSR2M4cnE=\nHost: %s:%d\r\n\r\n", hub_ip, command, ir.Id, hub_ip, hub_port)
		//path = fmt.Sprintf("/0?%x%02d=I=0", command, ir.Id)
		path = fmt.Sprintf("/3?%s=I=3", NewInsteonCommandPassThrough(ir.Id, command, 0xff).Format())
	}
	url := fmt.Sprintf("%s%s", host, path)
	println("URL:", url)
	client := http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		println("Failed to create request:", err.Error())
	}
	request.SetBasicAuth("Clifton8", "0BRGc8rq")
	resp, err := client.Do(request)
	if err != nil {
		println("InsteonResponder.sendCommand failed:", err.Error())
	}
	println("sendCommand.resp:")
	println("\tStatus:", resp.Status)
	println("\tHeader:", resp.Header)
}

func (ir *InsteonResponder) TurnOn() {
	ir.sendCommand(0x11)
}

func (ir *InsteonResponder) TurnOff() {
	ir.sendCommand(0x13)
}

func (ir *InsteonResponder) GetStatus() {
	host := fmt.Sprintf("http://%s:%d", hub_ip, hub_port)
	path := fmt.Sprintf("/3?0262%06x0F1900", ir.Id)
	url := fmt.Sprintf("%s%s", host, path)
	client := http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		println("Failed to create Status request:", err.Error())
	}
	request.SetBasicAuth("Clifton8", "0BRGc8rq")
	resp, err := client.Do(request)
	if err != nil {
		println("InsteonResponder.GetStatus failed:", err.Error())
	}
	println("GetStatus resp:")
	println("\tStatus:", resp.Status)
	println("\tHeader:", resp.Header)
}
