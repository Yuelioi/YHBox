//go:build windows

package main

import "golang.org/x/sys/windows"

const (
	messageBoxOK        = 0x00000000
	messageBoxIconError = 0x00000010
)

func showStartupError(message string) {
	text, textErr := windows.UTF16PtrFromString(message)
	caption, captionErr := windows.UTF16PtrFromString("Yotta 启动失败")
	if textErr != nil || captionErr != nil {
		return
	}
	_, _ = windows.MessageBox(0, text, caption, messageBoxOK|messageBoxIconError)
}
