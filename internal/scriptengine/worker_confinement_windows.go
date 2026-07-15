//go:build windows

package scriptengine

import (
	"errors"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	tokenIsAppContainer               = 29
	tokenSecurityAttributes           = 39
	tokenIsLessPrivilegedAppContainer = 46
	tokenSecurityAttributeUint64      = 0x0002
)

type tokenSecurityAttributesInformation struct {
	Version        uint16
	Reserved       uint16
	AttributeCount uint32
	Attributes     unsafe.Pointer
}

type tokenUnicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

type tokenSecurityAttributeV1 struct {
	Name       tokenUnicodeString
	ValueType  uint16
	Reserved   uint16
	Flags      uint32
	ValueCount uint32
	Values     unsafe.Pointer
}

var isProcessInJob = windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob")

func verifyWorkerConfinement() error {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return err
	}
	defer token.Close()
	isAppContainer, err := tokenBoolean(token, tokenIsAppContainer)
	if err != nil {
		return err
	}
	if !isAppContainer {
		return errWorkerNotAppContainer
	}
	isLPAC, err := tokenBoolean(token, tokenIsLessPrivilegedAppContainer)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		isLPAC, err = tokenHasNoAllApplicationPackagesAttribute(token)
	}
	if err != nil {
		return err
	}
	if !isLPAC {
		return errWorkerNotLPAC
	}
	process, err := windows.GetCurrentProcess()
	if err != nil {
		return errors.New("open current script worker process")
	}
	var inJob uint32
	result, _, callErr := isProcessInJob.Call(uintptr(process), 0, uintptr(unsafe.Pointer(&inJob)))
	if result == 0 {
		if callErr == nil || errors.Is(callErr, syscall.Errno(0)) {
			return errors.New("query script worker job membership")
		}
		return callErr
	}
	if inJob == 0 {
		return errWorkerNotInJob
	}
	return nil
}

func tokenHasNoAllApplicationPackagesAttribute(token windows.Token) (bool, error) {
	var size uint32
	err := windows.GetTokenInformation(token, tokenSecurityAttributes, nil, 0, &size)
	if !errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) || size == 0 {
		return false, err
	}
	buffer := make([]byte, size)
	if err := windows.GetTokenInformation(token, tokenSecurityAttributes, &buffer[0], size, &size); err != nil {
		return false, err
	}
	start := uintptr(unsafe.Pointer(&buffer[0]))
	end := start + uintptr(len(buffer))
	information := (*tokenSecurityAttributesInformation)(unsafe.Pointer(&buffer[0]))
	if information.Version != 1 || information.AttributeCount > 128 {
		return false, errors.New("invalid token security attributes header")
	}
	attributeBytes := uintptr(information.AttributeCount) * unsafe.Sizeof(tokenSecurityAttributeV1{})
	if information.AttributeCount != 0 && !pointerWithin(information.Attributes, attributeBytes, start, end) {
		return false, errors.New("token security attributes escape their buffer")
	}
	attributes := unsafe.Slice((*tokenSecurityAttributeV1)(information.Attributes), int(information.AttributeCount))
	for _, attribute := range attributes {
		name, err := boundedUnicodeString(attribute.Name, start, end)
		if err != nil {
			return false, err
		}
		if strings.EqualFold(name, "WIN://NOALLAPPPKG") {
			if attribute.Reserved != 0 || attribute.ValueType != tokenSecurityAttributeUint64 || attribute.ValueCount != 1 ||
				!pointerWithin(attribute.Values, unsafe.Sizeof(uint64(0)), start, end) {
				return false, errors.New("invalid WIN://NOALLAPPPKG token attribute")
			}
			if *(*uint64)(attribute.Values) == 0 {
				return false, nil
			}
			return true, nil
		}
	}
	return false, nil
}

func pointerWithin(pointer unsafe.Pointer, size, start, end uintptr) bool {
	address := uintptr(pointer)
	return pointer != nil && address >= start && address <= end && size <= end-address
}

func boundedUnicodeString(value tokenUnicodeString, start, end uintptr) (string, error) {
	if value.Length > value.MaximumLength || value.Length%2 != 0 || value.Length > 512 {
		return "", errors.New("token attribute name has invalid length")
	}
	if value.Length == 0 {
		return "", nil
	}
	if !pointerWithin(unsafe.Pointer(value.Buffer), uintptr(value.Length), start, end) {
		return "", errors.New("token attribute name escapes its buffer")
	}
	units := unsafe.Slice(value.Buffer, int(value.Length/2))
	return string(utf16.Decode(units)), nil
}

func tokenBoolean(token windows.Token, informationClass uint32) (bool, error) {
	var value uint32
	var returned uint32
	if err := windows.GetTokenInformation(
		token,
		informationClass,
		(*byte)(unsafe.Pointer(&value)),
		uint32(unsafe.Sizeof(value)),
		&returned,
	); err != nil {
		return false, err
	}
	if returned != 0 && returned != uint32(unsafe.Sizeof(value)) {
		return false, errors.New("unexpected script worker token information size")
	}
	return value != 0, nil
}
