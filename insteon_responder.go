package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

const hubIP = "192.168.0.100"
const hubPort = 25105
const hubUsername = "Clifton8"
const hubPassword = "0BRGc8rq"

const (
	InsteonResponderTypeGroup = iota
	InsteonResponderTypeDevice
)

func NewInsteonCommandPassThrough(id uint32, cmd1 uint8, cmd2 uint8) InsteonCommand {
	return InsteonCommand{
		PassThrough: 262,
		ID:          id,
		Flags:       0xF,
		Cmd1:        cmd1,
		Cmd2:        cmd2,
	}
}

func (ic InsteonCommand) Format() string {
	buf := fmt.Sprintf("%04d", ic.PassThrough)
	buf = fmt.Sprintf("%s%06X", buf, ic.ID)
	buf = fmt.Sprintf("%s%02X", buf, ic.Flags)
	buf = fmt.Sprintf("%s%02X", buf, ic.Cmd1)
	buf = fmt.Sprintf("%s%02X", buf, ic.Cmd2)
	return buf
}

type InsteonResponderType int

type InsteonCommand struct {
	PassThrough uint16 // 0262 for Pass-through to PLM
	ID          uint32 // ID for the command
	Flags       uint8  // FLags Byte (constant)
	Cmd1        uint8  // CMD1
	Cmd2        uint8  // CMD2
}

type InsteonResponder struct {
	mu    sync.RWMutex
	ID    uint32
	is_on bool
	Type  InsteonResponderType
}

func createAuthRequest(url string) *http.Request {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		println("Failed to create request:", err.Error())
	}
	request.SetBasicAuth(hubUsername, hubPassword)
	request.Close = true
	request.Header.Set("Connection", "close")
	return request
}

func (ir *InsteonResponder) sendCommand(command1 uint8, command2 uint8) {
	host := fmt.Sprintf("http://%s:%d", hubIP, hubPort)
	var path string
	switch ir.Type {
	case InsteonResponderTypeGroup:
		//path = fmt.Sprintf("/3?0262%06x0F%02xFF=I=3\n", ir.ID, command)
		path = fmt.Sprintf("/0?%02X%02d=I=0", command1, ir.ID)
	case InsteonResponderTypeDevice:
		//url = fmt.Sprintf("http://%s/0?%x%02d=I=0 HTTP/1.1\nAuthorization: Basic Q2xpZnRvbjg6MEJSR2M4cnE=\nHost: %s:%d\r\n\r\n", hubIP, command, ir.ID, hubIP, hubPort)
		//path = fmt.Sprintf("/0?%x%02d=I=0", command, ir.ID)
		path = fmt.Sprintf("/3?%s=I=3", NewInsteonCommandPassThrough(ir.ID, command1, command2).Format())
	}
	url := fmt.Sprintf("%s%s", host, path)

	request := createAuthRequest(url)

	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		println("InsteonResponder.sendCommand failed:", err.Error())
		return
	}
	//header, err := ioutil.ReadAll(resp.Header)
	//println("sendCommand.resp:")
	if resp.Status != "200 OK" {
		println("\tStatus:", resp.Status)
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			println("sendCommand: failed to ReadAll from resp.Body:", err.Error())
		}
		println("\tHeader:", resp.Header)
		println("\tBody:", string(body))
	}
	resp.Body.Close()
}

func (ir *InsteonResponder) Toggle() {
	ir.mu.Lock()
	if ir.is_on == true {
		ir.unsafeTurnOff()
		ir.is_on = false
	} else {
		ir.is_on = true
		ir.unsafeTurnOn()
	}
	ir.mu.Unlock()
}

func (ir *InsteonResponder) unsafeTurnOn() {
	println("Turning on")
	ir.sendCommand(0x11, 0xff)
}

func (ir *InsteonResponder) unsafeTurnOff() {
	println("Turning off")
	ir.sendCommand(0x13, 0xff)
}

func parseResponse(hub_response string) bool {
	// skip the <response><BS> junk at the front
	remainderString := hub_response[14 : len(hub_response)-1]
	// skip the command that's being mirrored back to me
	remainderString = remainderString[16 : len(remainderString)-1]
	// 06 - PLM says I got it
	println("remainderString:", remainderString)
	println("remainderString[0:1]:", remainderString[0:2])
	if remainderString[0:2] == "06" {
		println("Got it")
	} else {
		println("Ain't got it")
	}
	remainderString = remainderString[2 : len(remainderString)-1]

	// 0250 - PLM insteon received
	if remainderString[0:4] == "0250" {
		println("It's an Insteon Message")
	} else {
		println("I don't know what it is")
	}
	remainderString = remainderString[4 : len(remainderString)-1]

	// XXXXXX - Device ID
	println("Info is for Device ID:", remainderString[0:6])
	remainderString = remainderString[6 : len(remainderString)-1]

	// XXXXXX - PLM Device ID
	println("Info is from Device ID:", remainderString[0:6])
	remainderString = remainderString[6 : len(remainderString)-1]

	// XX - hop count
	println("Hop count is:", remainderString[0:2])
	remainderString = remainderString[2 : len(remainderString)-1]

	// XX - delta
	println("Delta:", remainderString[0:2])
	remainderString = remainderString[2 : len(remainderString)-1]

	// XX - power level. FF is all on
	println("Power level:", remainderString[0:2])
	if remainderString[0:2] != "00" {
		return true
	} else {
		return false
	}
}

func (ir *InsteonResponder) GetStatus() bool {
	ir.sendCommand(0x19, 0x00)

	time.Sleep(3 * time.Second)

	host := fmt.Sprintf("http://%s:%d", hubIP, hubPort)
	url := fmt.Sprintf("%s/buffstatus.xml", host)
	println("Getting buffstatus.xml")
	request := createAuthRequest(url)

	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		println("htt.Get returned err:", err.Error())
		return false
	}
	println("resp:", resp)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		println("Failed to read body:", err.Error())
	}

	resp.Body.Close()

	bodyString := string(bodyBytes)
	println("bodyString:", bodyString)
	return parseResponse(bodyString)
}

func (ir *InsteonResponder) UpdateStatus() bool {
	status_changed := false
	status := ir.GetStatus()
	ir.mu.RLock()
	old_status := ir.is_on
	ir.mu.RUnlock()
	if old_status != status {
		ir.mu.Lock()
		ir.is_on = status
		ir.mu.Unlock()
		status_changed = true
	}
	return status_changed
}
