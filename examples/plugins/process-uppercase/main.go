package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	pluginsdk "github.com/yottaapp/yotta/sdk/plugin/go"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != pluginsdk.ProcessGuestArgument {
		os.Exit(2)
	}
	guest, err := pluginsdk.NewGuest(os.Stdin, os.Stdout)
	if err != nil {
		os.Exit(3)
	}
	invocation, err := guest.ReceiveInvocation()
	if err != nil {
		os.Exit(4)
	}
	var input *pluginsdk.PortValue
	for _, candidate := range invocation.Inputs {
		if candidate.PortId == "value" {
			input = candidate
			break
		}
	}
	if input == nil {
		_ = guest.Fail("example.input_missing", "error", "value input is required")
		return
	}
	raw, err := pluginsdk.InlineJSON(input.ValueEnvelope)
	if err != nil {
		_ = guest.Fail("example.input_invalid", "error", "value input is invalid")
		return
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		_ = guest.Fail("example.input_invalid", "error", "value input is not a string")
		return
	}
	output, err := pluginsdk.ReplaceInlineJSON(input.ValueEnvelope, []byte(fmt.Sprintf("%q", strings.ToUpper(value))))
	if err != nil {
		_ = guest.Fail("example.output_invalid", "error", "result could not be encoded")
		return
	}
	_ = guest.Succeed(map[string][]byte{"result": output}, nil)
}
