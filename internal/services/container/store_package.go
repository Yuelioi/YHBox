package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yotta/internal/services/container/dependency"
)

const (
	packageFile      = "package.json"
	graphFile        = "graph.json"
	installationFile = "installation.json"
	lockFile         = "yotta-lock.json"
)

func readJSONFile[T any](path string) (T, error) {
	var out T
	b, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}

func writeJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func containerToPackageManifest(c Container) PackageManifest {
	version := c.Version
	if version == "" {
		version = "0.1.0"
	}
	keywords := append([]string(nil), c.Tags...)
	if len(keywords) == 0 {
		keywords = append([]string(nil), c.Keywords...)
	}
	return PackageManifest{
		SchemaVersion: PackageSchemaVersion,
		Kind:          PackageKindContainer,
		Name:          localPackageName(c.ID),
		DisplayName:   c.Name,
		Version:       version,
		Description:   c.Description,
		Keywords:      keywords,
		Category:      c.Category,
		Author:        c.Author,
		Publisher:     PackagePublisher{ID: "local", Name: "Local"},
		Yotta: PackageYotta{
			PackageID:  "pkg_" + c.ID,
			EntryGraph: graphFile,
			Publication: Publication{
				State:      PublicationDraft,
				Visibility: VisibilityPrivate,
			},
			Sources: []SourceRef{},
			Vars:    append([]VarDecl(nil), c.Vars...),
		},
	}
}

func localPackageName(id string) string {
	name := strings.ToLower(strings.ReplaceAll(id, "_", "-"))
	return "@local/" + name
}

func containerToInstallation(c Container, manifest PackageManifest) Installation {
	return Installation{
		SchemaVersion:    InstallationSchemaVersion,
		InstanceID:       c.ID,
		PackageID:        manifest.Yotta.PackageID,
		PackageName:      manifest.Name,
		InstalledVersion: manifest.Version,
		Display:          InstallationDisplay{},
		RuntimeOverrides: RuntimeOverrides{
			Hotkey:         stringPtrIfSet(c.Hotkey),
			InputBackend:   stringPtrIfSet(c.InputBackend),
			CaptureBackend: stringPtrIfSet(c.CaptureBackend),
			ScaleTolerance: floatPtrIfPositive(c.ScaleTolerance),
		},
		TargetBindings: map[string]TargetBinding{},
		AIBindings:     map[string]AIBinding{},
		Updates:        InstallationUpdates{AutoCheck: true},
		InstalledAt:    timeString(c.CreatedAt),
		UpdatedAt:      timeString(c.UpdatedAt),
	}
}

func aggregateContainer(manifest PackageManifest, graph Graph, installation Installation) Container {
	name := manifest.DisplayName
	if name == "" {
		name = manifest.Name
	}
	c := Container{
		SchemaVersion:  CurrentSchemaVersion,
		ID:             installation.InstanceID,
		Name:           name,
		Description:    manifest.Description,
		Tags:           append([]string(nil), manifest.Keywords...),
		PackageID:      manifest.Yotta.PackageID,
		PackageName:    manifest.Name,
		Version:        manifest.Version,
		Category:       manifest.Category,
		Keywords:       append([]string(nil), manifest.Keywords...),
		Author:         manifest.Author,
		Vars:           append([]VarDecl(nil), manifest.Yotta.Vars...),
		Graph:          hydrateGraphBindings(graph, installation),
		Hotkey:         stringValue(installation.RuntimeOverrides.Hotkey),
		InputBackend:   stringValue(installation.RuntimeOverrides.InputBackend),
		CaptureBackend: stringValue(installation.RuntimeOverrides.CaptureBackend),
		ScaleTolerance: floatValue(installation.RuntimeOverrides.ScaleTolerance),
		CreatedAt:      parseTime(installation.InstalledAt),
		UpdatedAt:      parseTime(installation.UpdatedAt),
	}
	if c.ID == "" {
		c.ID = strings.TrimPrefix(manifest.Yotta.PackageID, "pkg_")
	}
	return c
}

func buildContainerLock(manifest PackageManifest, graph Graph, subgraphs []Subgraph, generatedAt string) (YottaLock, error) {
	closure, err := dependencyClosure(graph, subgraphs)
	if err != nil {
		return YottaLock{}, err
	}
	return BuildYottaLock(manifest, graph, closure, generatedAt)
}

func splitPortableGraphBindings(graph Graph) (Graph, map[string]TargetBinding, map[string]AIBinding) {
	out := graph
	out.Nodes = append([]GraphNode(nil), graph.Nodes...)
	targetCount := countNodes(out.Nodes, "Win32WindowTarget") + countNodes(out.Nodes, "AndroidTarget")
	aiCount := countNodes(out.Nodes, "AI")
	targetBindings := map[string]TargetBinding{}
	aiBindings := map[string]AIBinding{}
	for i := range out.Nodes {
		n := out.Nodes[i]
		n.Config = cloneConfig(n.Config)
		switch n.Kind {
		case "Win32WindowTarget":
			slot := PinString(&n, "Target")
			if slot == "" {
				slot = defaultSlotName("game", "target_", n.ID, targetCount)
			}
			targetBindings[slot] = TargetBinding{
				Kind: "win32-window",
				Match: compactMap(map[string]any{
					"title":       PinString(&n, "Title"),
					"class":       PinString(&n, "Class"),
					"processName": PinString(&n, "ProcessName"),
					"titleMatch":  PinString(&n, "TitleMatch"),
				}),
			}
			removePins(&n, "Title", "Class", "ProcessName", "TitleMatch")
			SetPinValue(&n, "Target", slot)
		case "AndroidTarget":
			slot := PinString(&n, "Target")
			if slot == "" {
				slot = defaultSlotName("android", "target_", n.ID, targetCount)
			}
			width, _ := PinFloat(&n, "Width")
			height, _ := PinFloat(&n, "Height")
			targetBindings[slot] = TargetBinding{
				Kind: "android-adb",
				Match: compactMap(map[string]any{
					"serial": PinString(&n, "Serial"),
					"name":   PinString(&n, "Name"),
					"width":  width,
					"height": height,
				}),
			}
			removePins(&n, "Serial", "Name", "Width", "Height")
			SetPinValue(&n, "Target", slot)
		case "AI":
			conn := PinString(&n, "Connection")
			if conn != "" {
				slot := defaultSlotName("main", "ai_", n.ID, aiCount)
				aiBindings[slot] = AIBinding{ConnectionID: conn}
				removePins(&n, "Connection")
				SetPinValue(&n, "Connection", slot)
			}
		}
		out.Nodes[i] = n
	}
	return out, targetBindings, aiBindings
}

func hydrateGraphBindings(graph Graph, installation Installation) Graph {
	out := graph
	out.Nodes = append([]GraphNode(nil), graph.Nodes...)
	for i := range out.Nodes {
		n := out.Nodes[i]
		n.Config = cloneConfig(n.Config)
		switch n.Kind {
		case "Win32WindowTarget":
			if binding, ok := installation.TargetBindings[PinString(&n, "Target")]; ok {
				SetPinValue(&n, "Title", stringFromMap(binding.Match, "title"))
				SetPinValue(&n, "Class", stringFromMap(binding.Match, "class"))
				SetPinValue(&n, "ProcessName", stringFromMap(binding.Match, "processName"))
				SetPinValue(&n, "TitleMatch", stringFromMap(binding.Match, "titleMatch"))
			}
		case "AndroidTarget":
			if binding, ok := installation.TargetBindings[PinString(&n, "Target")]; ok {
				SetPinValue(&n, "Serial", stringFromMap(binding.Match, "serial"))
				SetPinValue(&n, "Name", stringFromMap(binding.Match, "name"))
				if v, ok := binding.Match["width"]; ok {
					SetPinValue(&n, "Width", v)
				}
				if v, ok := binding.Match["height"]; ok {
					SetPinValue(&n, "Height", v)
				}
			}
		case "AI":
			if binding, ok := installation.AIBindings[PinString(&n, "Connection")]; ok {
				SetPinValue(&n, "Connection", binding.ConnectionID)
			}
		}
		out.Nodes[i] = n
	}
	return out
}

func targetSlotsFromGraph(graph Graph) map[string]TargetSlot {
	out := map[string]TargetSlot{}
	for i := range graph.Nodes {
		n := &graph.Nodes[i]
		switch n.Kind {
		case "Win32WindowTarget":
			if slot := PinString(n, "Target"); slot != "" {
				out[slot] = TargetSlot{Kind: "win32-window"}
			}
		case "AndroidTarget":
			if slot := PinString(n, "Target"); slot != "" {
				out[slot] = TargetSlot{Kind: "android-adb"}
			}
		}
	}
	return out
}

func aiSlotsFromGraph(graph Graph) map[string]AISlot {
	out := map[string]AISlot{}
	for i := range graph.Nodes {
		n := &graph.Nodes[i]
		if n.Kind == "AI" {
			if slot := PinString(n, "Connection"); slot != "" {
				out[slot] = AISlot{}
			}
		}
	}
	return out
}

func depNodeInfos(nodes []GraphNode) []dependency.NodeInfo {
	out := make([]dependency.NodeInfo, 0, len(nodes))
	for i := range nodes {
		out = append(out, dependency.NodeInfo{Kind: nodes[i].Kind, Config: nodes[i].Config})
	}
	return out
}

func countNodes(nodes []GraphNode, kind string) int {
	n := 0
	for i := range nodes {
		if nodes[i].Kind == kind {
			n++
		}
	}
	return n
}

func defaultSlotName(single, prefix, id string, count int) string {
	if count <= 1 {
		return single
	}
	if id == "" {
		return prefix + "slot"
	}
	return prefix + id
}

func cloneConfig(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if k == "literal" {
			if lit, ok := v.(map[string]any); ok {
				cp := make(map[string]any, len(lit))
				for lk, lv := range lit {
					cp[lk] = lv
				}
				out[k] = cp
				continue
			}
		}
		out[k] = v
	}
	return out
}

func removePins(n *GraphNode, names ...string) {
	if n == nil || n.Config == nil {
		return
	}
	for _, name := range names {
		delete(n.Config, name)
	}
	if lit, ok := n.Config["literal"].(map[string]any); ok {
		for _, name := range names {
			delete(lit, name)
		}
	}
}

func compactMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		switch x := v.(type) {
		case string:
			if x != "" {
				out[k] = x
			}
		case float64:
			if x != 0 {
				out[k] = x
			}
		default:
			if v != nil {
				out[k] = v
			}
		}
	}
	return out
}

func stringFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func stringPtrIfSet(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}

func floatPtrIfPositive(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func floatValue(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func timeString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTime(v string) time.Time {
	t, _ := time.Parse(time.RFC3339, v)
	return t
}

func packagePath(dir string) string {
	return filepath.Join(dir, packageFile)
}
