package puppetcheck

import "testing"

func TestInterpretSummaryMarksRunAsFailedWhenFailuresExist(t *testing.T) {
	t.Parallel()

	summary := runSummary{
		Events: map[string]float64{
			"failure": 1,
			"success": 6,
			"total":   7,
		},
		Resources: map[string]float64{
			"failed": 0,
			"total":  20,
		},
	}

	interpreted := summary.interpret()
	if interpreted.RunSuccess != 0 {
		t.Fatalf("RunSuccess = %v, want 0", interpreted.RunSuccess)
	}
}

func TestInterpretSummaryMarksRunAsSuccessfulWhenNoFailures(t *testing.T) {
	t.Parallel()

	summary := runSummary{
		Events: map[string]float64{
			"failure": 0,
			"success": 6,
			"total":   7,
		},
		Resources: map[string]float64{
			"failed": 0,
			"total":  20,
		},
	}

	interpreted := summary.interpret()
	if interpreted.RunSuccess != 1 {
		t.Fatalf("RunSuccess = %v, want 1", interpreted.RunSuccess)
	}
}
