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
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricConfigMTime), prometheus.GaugeValue, snapshot.MTimeSeconds)
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricConfigReadErrors), prometheus.CounterValue, float64(snapshot.ReadErrorsTotal))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricConfigParseErrors), prometheus.CounterValue, float64(snapshot.ParseErrorsTotal))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricConfigScrapeDuration), prometheus.GaugeValue, snapshot.ScrapeDurationSeconds)
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
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentLockMTime), prometheus.GaugeValue, snapshot.MTimeSeconds)
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentLockReadErrors), prometheus.CounterValue, float64(snapshot.ReadErrorsTotal))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentLockParseErrors), prometheus.CounterValue, float64(snapshot.ParseErrorsTotal))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentLockScrapeDuration), prometheus.GaugeValue, snapshot.ScrapeDurationSeconds)
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentDisabled), prometheus.GaugeValue, framework.BoolFloat(snapshot.Valid && snapshot.Disabled), disabledMessage(snapshot))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentLockStateUp), prometheus.GaugeValue, framework.BoolFloat(snapshot.Up))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricAgentLockStateValid), prometheus.GaugeValue, framework.BoolFloat(snapshot.Valid))
}

func collectSummaryMetrics(ch chan<- prometheus.Metric, metrics featurekit.FeatureMetricDescriptors, snapshot puppetcheck.SummarySnapshot) {
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricLastRunSummaryMTime), prometheus.GaugeValue, snapshot.MTimeSeconds)
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricLastRunSummaryReadErrors), prometheus.CounterValue, float64(snapshot.ReadErrorsTotal))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricLastRunSummaryParseErrors), prometheus.CounterValue, float64(snapshot.ParseErrorsTotal))
	ch <- prometheus.MustNewConstMetric(metrics.Get(metricLastRunSummaryScrapeDuration), prometheus.GaugeValue, snapshot.ScrapeDurationSeconds)
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
