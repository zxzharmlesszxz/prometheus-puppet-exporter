package puppetcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"time"

	framework "github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

type Checker struct {
	Paths Paths

	configReadErrorsTotal   *atomic.Uint64
	configParseErrorsTotal  *atomic.Uint64
	lockReadErrorsTotal     *atomic.Uint64
	lockParseErrorsTotal    *atomic.Uint64
	summaryReadErrorsTotal  *atomic.Uint64
	summaryParseErrorsTotal *atomic.Uint64
}

func NewChecker(paths Paths) Checker {
	return Checker{
		Paths:                   paths,
		configReadErrorsTotal:   &atomic.Uint64{},
		configParseErrorsTotal:  &atomic.Uint64{},
		lockReadErrorsTotal:     &atomic.Uint64{},
		lockParseErrorsTotal:    &atomic.Uint64{},
		summaryReadErrorsTotal:  &atomic.Uint64{},
		summaryParseErrorsTotal: &atomic.Uint64{},
	}
}

func (c Checker) Snapshot(_ context.Context, now time.Time) Snapshot {
	config := c.snapshotConfig(c.Paths.Config)
	lock := c.snapshotLock(c.Paths.Lock)
	summary := c.snapshotSummary(c.Paths.Summary)

	return Snapshot{
		AttemptTime: now,
		Success:     config.Up && config.Valid && lock.Up && lock.Valid && summary.Up && summary.Valid,
		Config:      config,
		Lock:        lock,
		Summary:     summary,
	}
}

func (c Checker) snapshotConfig(path string) ConfigSnapshot {
	var config *ini.File
	readOK := false
	scrape := framework.FileScraper{
		ReadErrorsTotal:  c.configReadErrorsTotal,
		ParseErrorsTotal: c.configParseErrorsTotal,
	}.Scrape(path, func(content []byte) error {
		readOK = true
		var err error
		config, err = ini.LoadSources(ini.LoadOptions{}, content)
		return err
	})
	snapshot := ConfigSnapshot{
		Path:                  scrape.Path,
		MTimeSeconds:          scrape.MTimeSeconds,
		ReadErrorsTotal:       scrape.ReadErrorsTotal,
		ParseErrorsTotal:      scrape.ParseErrorsTotal,
		ScrapeDurationSeconds: scrape.ScrapeDurationSeconds,
		Err:                   scrape.Err,
	}
	if readOK {
		snapshot.Up = true
	}
	if !scrape.Up {
		return snapshot
	}
	snapshot.Server = config.Section("main").Key("server").String()
	snapshot.Certname = config.Section("main").Key("certname").String()
	snapshot.Environment = config.Section("main").Key("environment").String()
	snapshot.Valid = snapshot.Server != "" && snapshot.Certname != "" && snapshot.Environment != ""
	snapshot.Up = true
	return snapshot
}

func (c Checker) snapshotLock(path string) LockSnapshot {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return LockSnapshot{Path: path, Up: true, Valid: true}
	}

	var lock struct {
		DisabledMessage string `json:"disabled_message"`
	}
	readOK := false
	scrape := framework.FileScraper{
		ReadErrorsTotal:  c.lockReadErrorsTotal,
		ParseErrorsTotal: c.lockParseErrorsTotal,
	}.Scrape(path, func(content []byte) error {
		readOK = true
		return json.Unmarshal(content, &lock)
	})
	up := scrape.Up
	if readOK {
		up = true
	}
	return LockSnapshot{
		Path:                  scrape.Path,
		Up:                    up,
		Valid:                 scrape.Err == nil,
		MTimeSeconds:          scrape.MTimeSeconds,
		ReadErrorsTotal:       scrape.ReadErrorsTotal,
		ParseErrorsTotal:      scrape.ParseErrorsTotal,
		ScrapeDurationSeconds: scrape.ScrapeDurationSeconds,
		Err:                   scrape.Err,
		DisabledMessage:       lock.DisabledMessage,
	}
}

func (c Checker) snapshotSummary(path string) SummarySnapshot {
	var interpreted SummarySnapshot
	hasSummary := false
	readOK := false
	scrape := framework.FileScraper{
		ReadErrorsTotal:  c.summaryReadErrorsTotal,
		ParseErrorsTotal: c.summaryParseErrorsTotal,
	}.Scrape(path, func(content []byte) error {
		readOK = true
		var summary runSummary
		if err := yaml.NewDecoder(bytes.NewReader(content)).Decode(&summary); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if summary.empty() {
			return nil
		}
		interpreted = summary.interpret()
		hasSummary = true
		return nil
	})
	snapshot := SummarySnapshot{
		Path:                  scrape.Path,
		MTimeSeconds:          scrape.MTimeSeconds,
		ReadErrorsTotal:       scrape.ReadErrorsTotal,
		ParseErrorsTotal:      scrape.ParseErrorsTotal,
		ScrapeDurationSeconds: scrape.ScrapeDurationSeconds,
		Err:                   scrape.Err,
	}
	if readOK {
		snapshot.Up = true
	}
	if !scrape.Up || !hasSummary {
		return snapshot
	}
	interpreted.Path = snapshot.Path
	interpreted.Up = snapshot.Up
	interpreted.MTimeSeconds = snapshot.MTimeSeconds
	interpreted.ReadErrorsTotal = snapshot.ReadErrorsTotal
	interpreted.ParseErrorsTotal = snapshot.ParseErrorsTotal
	interpreted.ScrapeDurationSeconds = snapshot.ScrapeDurationSeconds
	return interpreted
}
