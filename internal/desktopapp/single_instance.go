package desktopapp

import (
	"crypto/sha256"
	"encoding/hex"
	"runtime"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/storage"
)

const singleInstanceNamespace = "com.yottaapp.yotta.profile."

func singleInstanceID(root string) (string, error) {
	roots, err := storage.Resolve(root)
	if err != nil {
		return "", err
	}
	identity := roots.Root
	if runtime.GOOS == "windows" {
		identity = strings.ToLower(identity)
	}
	digest := sha256.Sum256([]byte(identity))
	return singleInstanceNamespace + hex.EncodeToString(digest[:16]), nil
}

// mainWindowActivator keeps a second-launch request made during startup until
// the native main window is ready. Multiple early requests coalesce because
// they all ask for the same visible/focused state.
type mainWindowActivator struct {
	mu      sync.Mutex
	reveal  func()
	ready   bool
	pending bool
}

func (a *mainWindowActivator) request() {
	a.mu.Lock()
	if !a.ready || a.reveal == nil {
		a.pending = true
		a.mu.Unlock()
		return
	}
	reveal := a.reveal
	a.mu.Unlock()
	reveal()
}

func (a *mainWindowActivator) attach(reveal func()) {
	a.mu.Lock()
	a.reveal = reveal
	if !a.ready || !a.pending || reveal == nil {
		a.mu.Unlock()
		return
	}
	a.pending = false
	a.mu.Unlock()
	reveal()
}

func (a *mainWindowActivator) markReady() {
	a.mu.Lock()
	a.ready = true
	if !a.pending || a.reveal == nil {
		a.mu.Unlock()
		return
	}
	a.pending = false
	reveal := a.reveal
	a.mu.Unlock()
	reveal()
}

func handleMainWindowClosing(minimiseToTray bool, cancel, hide, quit func()) {
	cancel()
	if minimiseToTray {
		hide()
		return
	}
	quit()
}
