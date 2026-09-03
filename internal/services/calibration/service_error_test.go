package calibration

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestCalibrationProblemProjection(t *testing.T) {
	got := apperr.From(calibrationProblem("calibration.test", apperr.CategoryAdapter, true, errors.New("private")))
	if got.ID != "calibration.test" || got.Category != apperr.CategoryAdapter || !got.Retryable {
		t.Fatalf("problem = %#v", got)
	}
}
