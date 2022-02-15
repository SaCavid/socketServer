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
func (p *Protocol) Validate() error {

	if p.To == "" {
		return fmt.Errorf("machine id cant be null")
	}

	if len(p.To) > 30 {
		return fmt.Errorf("invalid machine name")
	}

	if len(p.Command) == 0 || len(p.Command) > 30 {
		return fmt.Errorf("wrong command format")
	}

	commands := []string{"CONSOLE", "REBOOT", "FORCEREBOOT", "SHUTDOWN", "SAFESHUTDOWN", "INSTANTOC", "SETFANS", "DOWNLOADWATTS", "RESTARTWATTS", "RESTART", "DIAG", "START", "NODERESTART", "RESTARTNODE", "STOP", "FLASH", "GETBIOS", "POWERCYCLE", "DIRECT"}
	for _, v := range commands {
		if p.Command == v {
			return nil
		}
	}

	return fmt.Errorf("malformed command")
}
