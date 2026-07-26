//go:build windows

package services

import (
	"errors"
	"reflect"
	"testing"
)

func TestApplyAutostartCreatesHighestPrivilegeLogonTask(t *testing.T) {
	var calls [][]string
	err := applyAutostart(true, func() (string, error) {
		return `E:\Program Files\Yotta\Yotta.exe`, nil
	}, func(args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{
		"/Create", "/TN", "Yotta", "/TR", `"E:\Program Files\Yotta\Yotta.exe"`,
		"/SC", "ONLOGON", "/IT", "/RL", "HIGHEST", "/F",
	}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected schtasks arguments: %#v", calls)
	}
}

func TestApplyAutostartDeleteIsIdempotent(t *testing.T) {
	calls := 0
	err := applyAutostart(false, nil, func(args ...string) ([]byte, error) {
		calls++
		return nil, errors.New("task not found")
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected one task query, got %d", calls)
	}
}

func TestApplyAutostartDeletesExistingTask(t *testing.T) {
	var calls [][]string
	err := applyAutostart(false, nil, func(args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"/Query", "/TN", "Yotta"},
		{"/Delete", "/TN", "Yotta", "/F"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected schtasks arguments: %#v", calls)
	}
}

func TestApplyAutostartRejectsUnsafeExecutablePath(t *testing.T) {
	err := applyAutostart(true, func() (string, error) {
		return "C:\\Yotta\nmalicious.exe", nil
	}, func(...string) ([]byte, error) {
		t.Fatal("runner should not be called")
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected unsafe executable path to be rejected")
	}
}
