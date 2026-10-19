package serial_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nickshiro/veil/infra/conf/serial"
)

func TestLoaderError(t *testing.T) {
	testCases := []struct {
		Input  string
		Output string
	}{
		{
			Input: `{
				"log": {

					0,
					"loglevel": "info"
				}
		}`,
			Output: "line 4 char 6",
		},
		{
			Input: `{
				"log": {

					"loglevel": "info",
				}
		}`,
			Output: "line 5 char 5",
		},
		{
			Input: `{
				"port": 1,
				"inbounds": [{
					"protocol": "test"
				}]
		}`,
			Output: "parse json config",
		},
		{
			Input: `{
				"inbounds": [{
					"port": 1,
					"listen": 0,
					"protocol": "test"
				}]
		}`,
			Output: "line 1 char 1",
		},
	}
	for _, testCase := range testCases {
		reader := bytes.NewReader([]byte(testCase.Input))
		_, err := serial.LoadJSONConfig(reader)
		if err == nil {
			t.Fatalf("expected error for %s", testCase.Input)
		}
		errString := err.Error()
		if !strings.Contains(errString, testCase.Output) {
			t.Error("unexpected output from json: ", testCase.Input, ". expected ", testCase.Output, ", but actually ", errString)
		}
	}
}
