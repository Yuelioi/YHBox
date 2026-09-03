package desktopapp

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/hotkey"
	"github.com/yottaapp/yotta/internal/services"
	"github.com/yottaapp/yotta/internal/services/workflow"
)

const launcherSlotKeyPrefix = "launcher.slot."

type launcherHotkeyController struct {
	mu       sync.Mutex
	registry *hotkey.HotkeyRegistry
	settings func() *services.Settings
	list     func() ([]workflow.SourceView, error)
	run      func(string)
}

func (c *launcherHotkeyController) refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()
	settings := c.settings()
	if settings == nil {
		_ = c.registry.ReplaceTransients(launcherSlotKeyPrefix, nil)
		return
	}
	views, err := c.list()
	if err != nil {
		return
	}
	available := make(map[string]workflow.SourceView, len(views))
	for _, view := range views {
		available[view.WorkflowID] = view
	}
	modifiers := strings.TrimSpace(settings.UI.LauncherSlotHotkeyModifiers)
	if modifiers == "" {
		modifiers = "Ctrl+Shift"
	}
	bindings := make([]hotkey.TransientBinding, 0, 9)
	ordinal := 0
	for _, block := range settings.UI.LauncherItems {
		if block.Type != "workflow" {
			continue
		}
		view, ok := available[block.WorkflowID]
		if !ok {
			continue
		}
		ordinal++
		if ordinal > 9 {
			break
		}
		workflowID := view.WorkflowID
		key := fmt.Sprintf("%s%d", launcherSlotKeyPrefix, ordinal)
		binding := fmt.Sprintf("%s+%d", modifiers, ordinal)
		bindings = append(bindings, hotkey.TransientBinding{
			Key: key, Label: "hotkeys.label.launcher_slot",
			LabelParams: map[string]string{"n": fmt.Sprint(ordinal), "name": view.Name},
			HotkeyStr:   binding, OnFire: func() { c.run(workflowID) },
		})
	}
	_ = c.registry.ReplaceTransients(launcherSlotKeyPrefix, bindings)
}

func (c *launcherHotkeyController) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clearLocked()
}

func (c *launcherHotkeyController) clearLocked() {
	_ = c.registry.ReplaceTransients(launcherSlotKeyPrefix, nil)
}

func syncWorkflowHotkeys(
	registry *hotkey.HotkeyRegistry,
	settings *services.Settings,
	views []workflow.SourceView,
	run func(string),
) {
	wanted := make(map[string]workflow.SourceView, len(views))
	for _, view := range views {
		wanted[view.WorkflowID] = view
		key := "action." + view.WorkflowID
		if _, exists := registry.Get(key); exists {
			registry.SyncLabel(key, "hotkeys.label.workflow")
			registry.SyncLabelParams(key, map[string]string{"name": view.Name})
			continue
		}
		workflowID := view.WorkflowID
		_ = registry.RegisterAction(
			key, "hotkeys.label.workflow",
			map[string]string{"name": view.Name}, strings.TrimSpace(settings.UI.WorkflowHotkeys[workflowID]),
			func() { run(workflowID) },
		)
	}
	for _, entry := range registry.List() {
		if entry.Source != hotkey.HotkeySourceAction || !strings.HasPrefix(entry.Key, "action.") {
			continue
		}
		if _, exists := wanted[strings.TrimPrefix(entry.Key, "action.")]; !exists {
			_ = registry.Unregister(entry.Key)
		}
	}
}
