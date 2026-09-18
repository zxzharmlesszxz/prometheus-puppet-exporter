package puppetcheck

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"time"
)

const maxFutureTimestampSkew = 5 * time.Minute

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

func (r runSummary) validate(now time.Time) error {
	if _, ok := r.Events["failure"]; !ok {
		return errors.New("summary events.failure is required")
	}
	if _, ok := r.Resources["failed"]; !ok {
		return errors.New("summary resources.failed is required")
	}
	if _, ok := r.Durations["last_run"]; !ok {
		return errors.New("summary time.last_run is required")
	}
	if err := validateMetricMap("events", r.Events); err != nil {
		return err
	}
	if err := validateMetricMap("resources", r.Resources); err != nil {
		return err
	}
	if err := validateMetricMap("time", r.Durations); err != nil {
		return err
	}
	if err := validateMetricValue("changes.total", r.Changes.Total); err != nil {
		return err
	}

	lastRun := r.Durations["last_run"]
	if lastRun == 0 {
		return errors.New("summary time.last_run must be positive")
	}
	latestAllowed := float64(now.Add(maxFutureTimestampSkew).Unix())
	if lastRun > latestAllowed {
		return fmt.Errorf("summary time.last_run must not be more than %s in the future", maxFutureTimestampSkew)
	}
	if catalogTimestamp, ok := r.Versions.CatalogTimestampSeconds(); ok && catalogTimestamp > latestAllowed {
		return fmt.Errorf("summary version.config must not be more than %s in the future", maxFutureTimestampSkew)
	}
	return nil
}

func validateMetricMap(section string, values map[string]float64) error {
	for name, value := range values {
		if err := validateMetricValue(section+"."+name, value); err != nil {
			return err
		}
	}
	return nil
}

func validateMetricValue(name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("summary %s must be finite", name)
	}
	if value < 0 {
		return fmt.Errorf("summary %s must not be negative", name)
	}
	return nil
}

func (r runSummary) interpret() SummarySnapshot {
	result := SummarySnapshot{
		Up:        true,
		Valid:     true,
		Changes:   r.Changes.Total,
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
	return r.Events["failure"] == 0 && r.Resources["failed"] == 0 && r.Resources["failed_to_restart"] == 0
}

func copyMetricMap(src map[string]float64) map[string]float64 {
	if len(src) == 0 {
		return map[string]float64{}
	}

	dst := make(map[string]float64, len(src))
	maps.Copy(dst, src)
	return dst
}
