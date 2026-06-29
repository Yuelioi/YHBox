package platform

import (
	"testing"
)

func TestTaskkillArgs_PID(t *testing.T) {
	args := taskkillArgs("1234")
	hasPID := false
	hasIM := false
	for _, a := range args {
		if a == "/PID" {
			hasPID = true
		}
		if a == "/IM" {
			hasIM = true
		}
	}
	if !hasPID {
		t.Errorf("taskkillArgs(\"1234\") = %v, want /PID", args)
	}
	if hasIM {
		t.Errorf("taskkillArgs(\"1234\") = %v, must not contain /IM", args)
	}
}

func TestTaskkillArgs_Name(t *testing.T) {
	args := taskkillArgs("chrome.exe")
	hasPID := false
	hasIM := false
	for _, a := range args {
		if a == "/PID" {
			hasPID = true
		}
		if a == "/IM" {
			hasIM = true
		}
	}
	if !hasIM {
		t.Errorf("taskkillArgs(\"chrome.exe\") = %v, want /IM", args)
	}
	if hasPID {
		t.Errorf("taskkillArgs(\"chrome.exe\") = %v, must not contain /PID", args)
	}
}

func TestTaskkillArgs_ExePathUsesImageName(t *testing.T) {
	args := taskkillArgs(`E:\adobe\Adobe After Effects 2022\Support Files\AfterFX.exe`)
	if len(args) != 3 {
		t.Fatalf("taskkillArgs(path) = %v, want 3 args", args)
	}
	if args[0] != "/F" || args[1] != "/IM" || args[2] != "AfterFX.exe" {
		t.Fatalf("taskkillArgs(path) = %v, want [/F /IM AfterFX.exe]", args)
	}
}
