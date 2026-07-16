package tools

import (
	"context"
	"fmt"
)

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

// PixelAt samples one exact installed target through its target tool adapter.
func (s *Service) PixelAt(targetSlot string) (PixelInfo, error) {
	if s.resolver == nil {
		return PixelInfo{}, fmt.Errorf("installed target resolver is unavailable")
	}
	tg, err := s.resolver.ResolveTarget(context.Background(), targetSlot)
	if err != nil {
		return PixelInfo{}, err
	}
	return s.targetTools.PixelAt(tg, PixelSampleRequest{TargetSlot: targetSlot})
}
