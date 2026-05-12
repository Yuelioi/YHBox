package cook

import "time"

type Config struct {
	Interval          time.Duration
	MissInterval      time.Duration
	MaxMiss           int
	ConfGate          float32
	HammerROI         [4]float64
	ClickHold         time.Duration
	ActivateDelay     time.Duration
	CursorSettleDelay time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		Interval:          1500 * time.Millisecond,
		MissInterval:      5 * time.Second,
		MaxMiss:           3,
		ConfGate:          0.45,
		HammerROI:         [4]float64{0.0, 0.20, 0.12, 0.40},
		ClickHold:         30 * time.Millisecond,
		ActivateDelay:     30 * time.Millisecond,
		CursorSettleDelay: 30 * time.Millisecond,
	}
}
