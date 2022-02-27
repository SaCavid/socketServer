package models

import (
	"fmt"
)

// Protocol main model for communication between modules
type Protocol struct {
	To      string
	Command string

	Error     bool
	ErrCode   uint32
	Msg       string
	AdminChan chan *Protocol
}

// Response simplified model for websocket responses
type Response struct {
	To      string
	Command string

	Error   bool
	ErrCode uint32
	Msg     string
}

// Validate checks protocol for valid information
func (p *Protocol) Validate() (Msg, error) {

	if p.To == "" {
		return InvalidWorker, fmt.Errorf("machine id cant be null")
	}

	if len(p.To) > 30 {
		return InvalidWorker, fmt.Errorf("invalid machine name")
	}

	if len(p.Command) == 0 || len(p.Command) > 30 {
		return MalformedCommand, fmt.Errorf("wrong command format")
	}

	commands := []string{"CONSOLE", "REBOOT", "FORCEREBOOT", "SHUTDOWN", "SAFESHUTDOWN", "INSTANTOC", "SETFANS", "DOWNLOADWATTS", "RESTARTWATTS", "RESTART", "DIAG", "START", "NODERESTART", "RESTARTNODE", "STOP", "FLASH", "GETBIOS", "POWERCYCLE", "DIRECT"}

	for _, v := range commands {
		if p.Command == v {
			return Ok, nil
		}
	}

	return MalformedCommand, fmt.Errorf("malformed command")
}
