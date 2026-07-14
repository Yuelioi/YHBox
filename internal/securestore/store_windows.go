//go:build windows

package securestore

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	credentialTypeGeneric         = 1
	credentialPersistLocalMachine = 2
)

var (
	advapi32        = windows.NewLazySystemDLL("advapi32.dll")
	procCredReadW   = advapi32.NewProc("CredReadW")
	procCredWriteW  = advapi32.NewProc("CredWriteW")
	procCredDeleteW = advapi32.NewProc("CredDeleteW")
	procCredFree    = advapi32.NewProc("CredFree")
)

type windowsCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

type windowsStore struct{}

func New() Store { return windowsStore{} }

func (windowsStore) Get(target string) (string, error) {
	targetName, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return "", fmt.Errorf("credential target: %w", err)
	}
	var credential *windowsCredential
	result, _, callErr := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetName)),
		credentialTypeGeneric,
		0,
		uintptr(unsafe.Pointer(&credential)),
	)
	if result == 0 {
		if errors.Is(callErr, syscall.ERROR_NOT_FOUND) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("read credential: %w", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(credential)))
	if credential == nil || credential.CredentialBlobSize == 0 {
		return "", nil
	}
	value := append([]byte(nil), unsafe.Slice(credential.CredentialBlob, int(credential.CredentialBlobSize))...)
	return string(value), nil
}

func (windowsStore) Set(target, value string) error {
	if value == "" {
		return windowsStore{}.Delete(target)
	}
	targetName, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("credential target: %w", err)
	}
	userName, err := windows.UTF16PtrFromString("Yotta")
	if err != nil {
		return fmt.Errorf("credential username: %w", err)
	}
	blob := []byte(value)
	credential := windowsCredential{
		Type:               credentialTypeGeneric,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     &blob[0],
		Persist:            credentialPersistLocalMachine,
		UserName:           userName,
	}
	result, _, callErr := procCredWriteW.Call(uintptr(unsafe.Pointer(&credential)), 0)
	if result == 0 {
		return fmt.Errorf("write credential: %w", callErr)
	}
	return nil
}

func (windowsStore) Delete(target string) error {
	targetName, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("credential target: %w", err)
	}
	result, _, callErr := procCredDeleteW.Call(
		uintptr(unsafe.Pointer(targetName)),
		credentialTypeGeneric,
		0,
	)
	if result == 0 && !errors.Is(callErr, syscall.ERROR_NOT_FOUND) {
		return fmt.Errorf("delete credential: %w", callErr)
	}
	return nil
}
