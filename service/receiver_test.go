package service

import (
	"fmt"
	"testing"
)

type TestLogin struct {
	Login string
	B     bool
}

// s0m4k3y.Worker001|SHUTDOWN
func TestValidateLogin(t *testing.T) {
	samples := []TestLogin{
		{
			Login: "",
			B:     true,
		},
		{
			Login: "s0m4k3y.Worker001",
			B:     false,
		},
	}

	for k, v := range samples {
		testName := fmt.Sprintf("%d,%s", k, v.Login)
		t.Run(testName, func(t *testing.T) {
			_, _, _, err := ValidateLogin([]byte(v.Login))
			if err != nil && !v.B {
				t.Errorf("%s", err.Error())
			}
		})
	}

}
