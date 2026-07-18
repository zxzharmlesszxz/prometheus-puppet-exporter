package puppetcheck

import (
	"strconv"
	"time"
)

type Paths struct {
	Config  string
	Lock    string
	Summary string
}

type Snapshot struct {
	AttemptTime time.Time
	Success     bool
	Config      ConfigSnapshot
	Lock        LockSnapshot
	Summary     SummarySnapshot
}

type ConfigSnapshot struct {
	Path                  string
	Up                    bool
	Valid                 bool
	Server                string
	Environment           string
	Certname              string
	MTimeSeconds          float64
	ReadErrorsTotal       uint64
	ParseErrorsTotal      uint64
	ScrapeDurationSeconds float64
	Err                   error
}

type LockSnapshot struct {
	Path                  string
	Up                    bool
	Valid                 bool
	Disabled              bool
	DisabledMessage       string
	MTimeSeconds          float64
	ReadErrorsTotal       uint64
	ParseErrorsTotal      uint64
	ScrapeDurationSeconds float64
	Err                   error
}

type SummarySnapshot struct {
	Path                  string
	Up                    bool
	Valid                 bool
	Versions              Versions
	RunSuccess            float64
	Resources             map[string]float64
	Events                map[string]float64
	Durations             map[string]float64
	LastRunTimestamp      float64
	MTimeSeconds          float64
	ReadErrorsTotal       uint64
	ParseErrorsTotal      uint64
	ScrapeDurationSeconds float64
	Err                   error
}

type Versions struct {
	Catalog string `yaml:"config"`
	Puppet  string `yaml:"puppet"`
}

func (v Versions) CatalogTimestampSeconds() (float64, bool) {
	timestamp, err := strconv.ParseInt(v.Catalog, 10, 64)
	return float64(timestamp), err == nil && timestamp > 0
}
