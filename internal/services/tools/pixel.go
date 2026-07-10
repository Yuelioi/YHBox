package tools

import "github.com/yottaapp/yotta/internal/automation/target"

// PixelInfo is a sampled target pixel in client coordinates.
type PixelInfo struct {
	OK      bool   `json:"ok"`
	ClientX int    `json:"clientX"`
	ClientY int    `json:"clientY"`
	R       int    `json:"r"`
	G       int    `json:"g"`
	B       int    `json:"b"`
	H       int    `json:"h"`
	S       int    `json:"s"`
	V       int    `json:"v"`
	Hex     string `json:"hex"`
}

// PixelAt samples the active editor target through its target tool adapter.
func (s *Service) PixelAt(containerID, nodeID string) (PixelInfo, error) {
	targetKind := target.KindWin32Window
	if s.resolver != nil {
		resolved, err := s.resolver.ResolveEditorTargetKindForNode(containerID, nodeID)
		if err != nil {
			return PixelInfo{}, err
		}
		if resolved != "" {
			targetKind = resolved
		}
	}
	return s.targetTools.PixelAt(targetKind, PixelSampleRequest{ContainerID: containerID, NodeID: nodeID})
}
