//go:build windows

package input

import "fmt"

// NewBackend constructs a Windows input adapter by its configuration name.
func NewBackend(name string) (Backend, error) {
	switch name {
	case "", "postmessage":
		return newPostMessageBackend(), nil
	case "sendinput":
		return newSendInputBackend(), nil
	default:
		return nil, fmt.Errorf("unknown input backend %q (supported: postmessage, sendinput)", name)
	}
}
