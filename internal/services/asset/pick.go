// internal/services/asset/pick.go
package asset

// PickVariant 从 guid 对应记录的 Variants 中挑最合适的一档。
// 精确分辨率命中优先；miss 时按长边比距最近的档兜底（调用方据 v.Resolution 自算缩放比）。
// 算法 = 精确分辨率命中优先, 否则长边比最近的档 (跨分辨率缩放兜底候选)。
func (s *Store) PickVariant(guid string, frameW, frameH int) (Variant, bool, error) {
	rec, ok, err := s.Record(guid)
	if err != nil {
		return Variant{}, false, err
	}
	if !ok {
		return Variant{}, false, nil
	}
	if len(rec.Variants) == 0 {
		return Variant{}, false, nil
	}

	// 精确命中优先。
	for _, v := range rec.Variants {
		if v.Resolution[0] == frameW && v.Resolution[1] == frameH {
			return v, true, nil
		}
	}

	// miss → 挑长边比最接近的一档（跨分辨率缩放兜底候选）。调用方据 v.Resolution 算缩放比。
	frameLong := frameW
	if frameH > frameLong {
		frameLong = frameH
	}
	var best Variant
	bestScore := 0.0
	found := false
	for _, v := range rec.Variants {
		vl := v.Resolution[0]
		if v.Resolution[1] > vl {
			vl = v.Resolution[1]
		}
		if vl <= 0 || frameLong <= 0 {
			continue
		}
		ratio := float64(frameLong) / float64(vl)
		if ratio < 1.0 {
			ratio = 1.0 / ratio // 对称比距：缩小/放大同等看待。
		}
		if !found || ratio < bestScore {
			best, bestScore, found = v, ratio, true
		}
	}
	return best, found, nil
}
