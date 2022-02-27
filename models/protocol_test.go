package models

import (
	"fmt"
	"testing"
)

type TestProtocol struct {
	Protocol
	B bool
}

func TestProtocol_Validate(t *testing.T) {

	samples := []TestProtocol{
		{
			Protocol: Protocol{
				To:        "Worker001",
				Command:   "Status", // wrong command
				Error:     false,
				ErrCode:   0,
				Msg:       "",
				AdminChan: nil,
			},
			B: true, // must be true if its wrong data
		},
		{
			Protocol: Protocol{
				To:        "Worker001",
				Command:   "REBOOT", // wrong command
				Error:     false,
				ErrCode:   0,
				Msg:       "",
				AdminChan: nil,
			},
			B: false, // must be true if its wrong data
		},
	}

	for _, v := range samples {
		testName := fmt.Sprintf("%s,%s", v.To, v.Command)
		t.Run(testName, func(t *testing.T) {
			_, err := v.Validate()
			if err != nil && !v.B {
				t.Errorf("%s", err.Error())
			}
		})
	}
}
