package desktopapp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSingleInstanceIDIsStablePerStorageRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "Profile")

	first, err := singleInstanceID(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := singleInstanceID(filepath.Join(root, "."))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("equivalent roots produced different instance IDs: %q != %q", first, second)
	}
	if !strings.HasPrefix(first, "com.yottaapp.yotta.profile.") {
		t.Fatalf("instance ID %q does not use the Yotta profile namespace", first)
	}

	other, err := singleInstanceID(filepath.Join(parent, "Other"))
	if err != nil {
		t.Fatal(err)
	}
	if first == other {
		t.Fatalf("different roots produced the same instance ID %q", first)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("resolving the instance ID created or inspected the profile unexpectedly: %v", err)
	}

	if runtime.GOOS == "windows" {
		folded, err := singleInstanceID(strings.ToUpper(root))
		if err != nil {
			t.Fatal(err)
		}
		if first != folded {
			t.Fatalf("Windows path casing produced different instance IDs: %q != %q", first, folded)
		}
	}
}

func TestMainWindowActivatorQueuesUntilApplicationIsReady(t *testing.T) {
	var reveals int
	activator := &mainWindowActivator{}

	activator.request()
	activator.request()
	activator.attach(func() { reveals++ })
	if reveals != 0 {
		t.Fatalf("main window revealed before application ready: %d", reveals)
	}

	activator.markReady()
	if reveals != 1 {
		t.Fatalf("queued activation reveals = %d, want 1", reveals)
	}

	activator.request()
	if reveals != 2 {
		t.Fatalf("ready activation reveals = %d, want 2", reveals)
	}
}

func TestHandleMainWindowClosingSeparatesHideAndQuit(t *testing.T) {
	t.Run("minimise to tray", func(t *testing.T) {
		cancelled, hidden, quit := 0, 0, 0
		handleMainWindowClosing(true,
			func() { cancelled++ },
			func() { hidden++ },
			func() { quit++ },
		)
		if cancelled != 1 || hidden != 1 || quit != 0 {
			t.Fatalf("actions = cancel:%d hide:%d quit:%d", cancelled, hidden, quit)
		}
	})

	t.Run("quit application", func(t *testing.T) {
		cancelled, hidden, quit := 0, 0, 0
		handleMainWindowClosing(false,
			func() { cancelled++ },
			func() { hidden++ },
			func() { quit++ },
		)
		if cancelled != 1 || hidden != 0 || quit != 1 {
			t.Fatalf("actions = cancel:%d hide:%d quit:%d", cancelled, hidden, quit)
		}
	})
}
