package runtime

import "yotta/internal/services/container"

func speedUpStateFixtureTimingForTest(sg *container.Subgraph) {
	for i := range sg.Graph.Nodes {
		literal, ok := sg.Graph.Nodes[i].Config["literal"].(map[string]any)
		if !ok {
			continue
		}
		capTimingLiteralForTest(literal, "Duration", 1)
		capTimingLiteralForTest(literal, "DurationMs", 1)
		capTimingLiteralForTest(literal, "TimeoutMs", 1)
		capTimingLiteralForTest(literal, "CooldownMs", 1)
		capTimingLiteralForTest(literal, "IntervalMs", 1)
	}
}

func capTimingLiteralForTest(literal map[string]any, key string, value float64) {
	if _, ok := literal[key]; ok {
		literal[key] = value
	}
}
