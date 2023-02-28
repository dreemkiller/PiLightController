package intsteon_controller

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const hub_ip = "192.168.0.100"
const hub_port = 25105
const hub_username = "Clifton8"
const hub_password = "0BRGc8rq"

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

func createAuthRequest(url string) *http.Request {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		println("Failed to create request:", err.Error())
	}
	request.SetBasicAuth(hub_username, hub_password)
	return request
}
func (ir *InsteonResponder) sendCommand(command1 uint8, command2 uint8) {
	host := fmt.Sprintf("http://%s:%d", hub_ip, hub_port)
	var path string
	switch ir.Type {
	case InsteonResponderTypeGroup:
		//path = fmt.Sprintf("/3?0262%06x0F%02xFF=I=3\n", ir.Id, command)
		path = fmt.Sprintf("/0?%02X%02d=I=0", command1, ir.Id)
	case InsteonResponderTypeDevice:
		//url = fmt.Sprintf("http://%s/0?%x%02d=I=0 HTTP/1.1\nAuthorization: Basic Q2xpZnRvbjg6MEJSR2M4cnE=\nHost: %s:%d\r\n\r\n", hub_ip, command, ir.Id, hub_ip, hub_port)
		//path = fmt.Sprintf("/0?%x%02d=I=0", command, ir.Id)
		path = fmt.Sprintf("/3?%s=I=3", NewInsteonCommandPassThrough(ir.Id, command1, command2).Format())
	}
	url := fmt.Sprintf("%s%s", host, path)
	println("URL:", url)

	request := createAuthRequest(url)

	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		println("InsteonResponder.sendCommand failed:", err.Error())
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		println("sendCommand: failed to read body:", err.Error())
		return
	}
	// header, _ := ioutil.ReadAll(resp.Header)
	println("sendCommand.resp:")
	println("\tStatus:", resp.Status)
	println("\tHeader:", resp.Header)
	println("\tBody:", string(body))
}

func (ir *InsteonResponder) TurnOn() {
	ir.sendCommand(0x11, 0xff)
}

func (ir *InsteonResponder) TurnOff() {
	ir.sendCommand(0x13, 0xff)
}

func parseResponse(hub_response string) {
	// skip the <response><BS> junk at the front
	remainder_string := hub_response[14 : len(hub_response)-1]
	// skip the command that's being mirrored back to me
	remainder_string = remainder_string[16 : len(remainder_string)-1]
	// 06 - PLM says I got it
	println("remainder_string:", remainder_string)
	println("remainder_string[0:1]:", remainder_string[0:2])
	if remainder_string[0:2] == "06" {
		println("Got it")
	} else {
		println("Ain't got it")
	}
	remainder_string = remainder_string[2 : len(remainder_string)-1]

	// 0250 - PLM insteon received
	if remainder_string[0:4] == "0250" {
		println("It's an Insteon Message")
	} else {
		println("I don't know what it is")
	}
	remainder_string = remainder_string[4 : len(remainder_string)-1]

	// XXXXXX - Device ID
	println("Info is for Device ID:", remainder_string[0:6])
	remainder_string = remainder_string[6 : len(remainder_string)-1]

	// XXXXXX - PLM Device ID
	println("Info is from Device ID:", remainder_string[0:6])
	remainder_string = remainder_string[6 : len(remainder_string)-1]

	// XX - hop count
	println("Hop count is:", remainder_string[0:2])
	remainder_string = remainder_string[2 : len(remainder_string)-1]

	// XX - delta
	println("Delta:", remainder_string[0:2])
	remainder_string = remainder_string[2 : len(remainder_string)-1]

	// XX - power level. FF is all on
	println("Power level:", remainder_string[0:2])
}

func (ir *InsteonResponder) GetStatus() {
	ir.sendCommand(0x19, 0x00)

	time.Sleep(3 * time.Second)

	host := fmt.Sprintf("http://%s:%d", hub_ip, hub_port)
	url := fmt.Sprintf("%s/buffstatus.xml", host)
	println("Getting buffstatus.xml")
	request := createAuthRequest(url)

	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		println("htt.Get returned err:", err.Error())
		return
	}
	println("resp:", resp)
	body_bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		println("Failed to read body:", err.Error())
		return
	}

	body_string := string(body_bytes)
	println("body_string:", body_string)
	parseResponse(body_string)
}
