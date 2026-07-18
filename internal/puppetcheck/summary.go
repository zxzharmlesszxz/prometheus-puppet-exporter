package puppetcheck

import "errors"

type runSummary struct {
	Changes   Changes            `yaml:"changes"`
	Events    map[string]float64 `yaml:"events"`
	Resources map[string]float64 `yaml:"resources"`
	Durations map[string]float64 `yaml:"time"`
	Versions  Versions           `yaml:"version"`
}

type Changes struct {
	Total float64 `yaml:"total"`
}

func (r runSummary) validate() error {
	if _, ok := r.Events["failure"]; !ok {
		return errors.New("summary events.failure is required")
	}
	if _, ok := r.Resources["failed"]; !ok {
		return errors.New("summary resources.failed is required")
	}
	if _, ok := r.Durations["last_run"]; !ok {
		return errors.New("summary time.last_run is required")
	}
	return nil
}

func (r runSummary) interpret() SummarySnapshot {
	result := SummarySnapshot{
		Up:        true,
		Valid:     true,
		Durations: copyMetricMap(r.Durations),
		Versions:  r.Versions,
		Resources: copyMetricMap(r.Resources),
		Events:    copyMetricMap(r.Events),
	}
	result.LastRunTimestamp = result.Durations["last_run"]
	delete(result.Durations, "last_run")
	if r.success() {
		result.RunSuccess = 1
	}
	return result
}

func (r runSummary) success() bool {
	return r.Events["failure"] == 0 && r.Resources["failed"] == 0
}

func copyMetricMap(src map[string]float64) map[string]float64 {
	if len(src) == 0 {
		return map[string]float64{}
	}

	dst := make(map[string]float64, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
