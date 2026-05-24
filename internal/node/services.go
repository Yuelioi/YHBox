// internal/node/services.go
// Phase 1 stub services. Phase 4 由 main.go 注入真实 backend (wire from wire_container.go).
package node

import "log"

type stdoutLogService struct{}

func (stdoutLogService) Debug(format string, args ...any) { log.Printf("[DEBUG] "+format, args...) }
func (stdoutLogService) Info(format string, args ...any)  { log.Printf("[INFO] "+format, args...) }
func (stdoutLogService) Warn(format string, args ...any)  { log.Printf("[WARN] "+format, args...) }

// DefaultLogService stdout-based, test 用. main.go 注入 zerolog-based 替换.
func DefaultLogService() LogService { return stdoutLogService{} }

// stubVisionService 测试用. 任何 key 都返 nil/0/nil (always miss).
type stubVisionService struct{}

func (stubVisionService) Match(key string, threshold float64) (*Point, float64, error) {
	return nil, 0, nil
}

// StubVisionService — Phase 1+2 test 用. Phase 4 main.go 注入真 wire_container.go::templateMatcherAdapter.
func StubVisionService() VisionService { return stubVisionService{} }
