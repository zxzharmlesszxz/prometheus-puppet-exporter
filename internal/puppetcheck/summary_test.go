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

func TestInterpretSummaryMarksRunAsFailedWhenRestartFails(t *testing.T) {
	t.Parallel()

	summary := runSummary{
		Events: map[string]float64{"failure": 0},
		Resources: map[string]float64{
			"failed":            0,
			"failed_to_restart": 1,
		},
	}

	interpreted := summary.interpret()
	if interpreted.RunSuccess != 0 {
		t.Fatalf("RunSuccess = %v, want 0", interpreted.RunSuccess)
	}
}

func TestCatalogTimestampSeconds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		catalog string
		want    float64
		wantOK  bool
	}{
		{name: "unix timestamp", catalog: "1774217222", want: 1774217222, wantOK: true},
		{name: "custom version", catalog: "release-42", want: 0, wantOK: false},
		{name: "zero", catalog: "0", want: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := (Versions{Catalog: tt.catalog}).CatalogTimestampSeconds()
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("CatalogTimestampSeconds() = %v, %v; want %v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
