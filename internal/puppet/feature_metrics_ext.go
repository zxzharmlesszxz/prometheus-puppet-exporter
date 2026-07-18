package puppet

import (
	"log/slog"
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	framework "github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter"
	"github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter/featurekit"

	"github.com/zxzharmlesszxz/prometheus-puppet-exporter/internal/puppetcheck"
)

func NewFeatureMetricHandlers() featurekit.FeatureMetricHandlers[Snapshot] {
	return featurekit.FeatureMetricHandlers[Snapshot]{
		Collect:  CollectFeatureMetrics,
		LogError: LogFeatureSnapshotError,
	}
}

func CollectFeatureMetrics(ctx featurekit.FeatureMetricsContext[Snapshot], ch chan<- prometheus.Metric, snapshot Snapshot, _ time.Time) {
	collectConfigMetrics(ch, ctx.Descriptors, snapshot.puppet.Config)
	collectLockMetrics(ch, ctx.Descriptors, snapshot.puppet.Lock)
	collectSummaryMetrics(ch, ctx.Descriptors, snapshot.puppet.Summary)
}

func LogFeatureSnapshotError(_ featurekit.FeatureMetricsContext[Snapshot], logger *slog.Logger, snapshot Snapshot) {
	if snapshot.puppet.Config.Err != nil {
		logger.Error("puppet_open_config_failed", "err", snapshot.puppet.Config.Err)
	}
	if snapshot.puppet.Lock.Err != nil {
		logger.Error("puppet_read_lock_failed", "err", snapshot.puppet.Lock.Err)
	}
	if snapshot.puppet.Summary.Err != nil {
		logger.Error("puppet_read_run_summary_failed", "err", snapshot.puppet.Summary.Err)
	}
}

func collectConfigMetrics(ch chan<- prometheus.Metric, metrics featurekit.FeatureMetricDescriptors, snapshot puppetcheck.ConfigSnapshot) {
	collectSourceMetrics(ch, metrics, configSourceMetricIDs, snapshot.MTimeSeconds, snapshot.ReadErrorsTotal, snapshot.ParseErrorsTotal, snapshot.ScrapeDurationSeconds)
	if snapshot.Up && snapshot.Valid {
		ch <- prometheus.MustNewConstMetric(
			metrics.Get(metricConfig),
			prometheus.GaugeValue,
			1,
			snapshot.Server,
			snapshot.Environment,
			snapshot.Certname,
		)
	}
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricConfigUp), prometheus.GaugeValue, framework.BoolFloat(snapshot.Up))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricConfigValid), prometheus.GaugeValue, framework.BoolFloat(snapshot.Up && snapshot.Valid))
}

func collectLockMetrics(ch chan<- prometheus.Metric, metrics featurekit.FeatureMetricDescriptors, snapshot puppetcheck.LockSnapshot) {
	collectSourceMetrics(ch, metrics, lockSourceMetricIDs, snapshot.MTimeSeconds, snapshot.ReadErrorsTotal, snapshot.ParseErrorsTotal, snapshot.ScrapeDurationSeconds)
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentDisabled), prometheus.GaugeValue, framework.BoolFloat(snapshot.Valid && snapshot.Disabled), disabledMessage(snapshot))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentLockStateUp), prometheus.GaugeValue, framework.BoolFloat(snapshot.Up))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentLockStateValid), prometheus.GaugeValue, framework.BoolFloat(snapshot.Valid))
}

func collectSummaryMetrics(ch chan<- prometheus.Metric, metrics featurekit.FeatureMetricDescriptors, snapshot puppetcheck.SummarySnapshot) {
	collectSourceMetrics(ch, metrics, summarySourceMetricIDs, snapshot.MTimeSeconds, snapshot.ReadErrorsTotal, snapshot.ParseErrorsTotal, snapshot.ScrapeDurationSeconds)
	if snapshot.Valid {
		ch <- prometheus.MustNewConstMetric(metrics.Get(metricVersionInfo), prometheus.GaugeValue, 1, snapshot.Versions.Catalog, "CatalogVersion")
		ch <- prometheus.MustNewConstMetric(metrics.Get(metricVersionInfo), prometheus.GaugeValue, 1, snapshot.Versions.Puppet, "PuppetVersion")
		if catalogTimestamp, ok := snapshot.Versions.CatalogTimestampSeconds(); ok {
			ch <- prometheus.MustNewConstMetric(metrics.Get(metricCatalogVersionTimestamp), prometheus.GaugeValue, catalogTimestamp)
		}
		ch <- prometheus.MustNewConstMetric(metrics.Get(metricLastRunSuccess), prometheus.GaugeValue, snapshot.RunSuccess)
		ch <- prometheus.MustNewConstMetric(metrics.Get(metricLastRunTimestamp), prometheus.GaugeValue, snapshot.LastRunTimestamp)
		collectNamedMetrics(ch, metrics.Get(metricLastRunDuration), snapshot.Durations)
		collectNamedMetrics(ch, metrics.Get(metricResources), snapshot.Resources)
		collectNamedMetrics(ch, metrics.Get(metricEvents), snapshot.Events)
	}
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricLastRunSummaryUp), prometheus.GaugeValue, framework.BoolFloat(snapshot.Up))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricLastRunSummaryValid), prometheus.GaugeValue, framework.BoolFloat(snapshot.Valid))
}

type sourceMetricIDs struct {
	mtime          string
	readErrors     string
	parseErrors    string
	scrapeDuration string
}

var (
	configSourceMetricIDs  = newSourceMetricIDs(metricConfigMTime, metricConfigReadErrors, metricConfigParseErrors, metricConfigScrapeDuration)
	lockSourceMetricIDs    = newSourceMetricIDs(metricAgentLockMTime, metricAgentLockReadErrors, metricAgentLockParseErrors, metricAgentLockScrapeDuration)
	summarySourceMetricIDs = newSourceMetricIDs(metricLastRunSummaryMTime, metricLastRunSummaryReadErrors, metricLastRunSummaryParseErrors, metricLastRunSummaryScrapeDuration)
)

func newSourceMetricIDs(mtime, readErrors, parseErrors, scrapeDuration string) sourceMetricIDs {
	return sourceMetricIDs{
		mtime:          mtime,
		readErrors:     readErrors,
		parseErrors:    parseErrors,
		scrapeDuration: scrapeDuration,
	}
}

func collectSourceMetrics(
	ch chan<- prometheus.Metric,
	metrics featurekit.FeatureMetricDescriptors,
	ids sourceMetricIDs,
	mtime float64,
	readErrors uint64,
	parseErrors uint64,
	scrapeDuration float64,
) {
	ch <- prometheus.MustNewConstMetric(metrics.Get(ids.mtime), prometheus.GaugeValue, mtime)
	ch <- prometheus.MustNewConstMetric(metrics.Get(ids.readErrors), prometheus.CounterValue, float64(readErrors))
	ch <- prometheus.MustNewConstMetric(metrics.Get(ids.parseErrors), prometheus.CounterValue, float64(parseErrors))
	ch <- prometheus.MustNewConstMetric(metrics.Get(ids.scrapeDuration), prometheus.GaugeValue, scrapeDuration)
}

func collectNamedMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, values map[string]float64) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, values[key], key)
	}
}

func disabledMessage(snapshot puppetcheck.LockSnapshot) string {
	if !snapshot.Valid {
		return ""
	}
	return snapshot.DisabledMessage
}
