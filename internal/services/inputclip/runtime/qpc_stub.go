//go:build !windows

package runtime

import "time"

func QPCMicros() uint64 {
	return uint64(time.Now().UnixMicro())
}

func TimerSetPrecision1ms() {}
func TimerResetPrecision()  {}
