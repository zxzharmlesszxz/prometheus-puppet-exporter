package puppet

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter/exportertest"
	"github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter/exportertest/featuretest"
	"github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter/featurekit"
	"github.com/zxzharmlesszxz/prometheus-puppet-exporter/internal/puppetcheck"
)

func TestFeatureContract(t *testing.T) {
	t.Parallel()

	suite := NewFeatureTestSuite(NewFeatureTestSpec())
	RegisterFeatureTests(suite)
	suite.RunTests(t)
}

func TestValidateFeatureConfigRejectsBlankPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
	}{
		{name: "config path", config: Config{ConfigPath: " \t", LockPath: DefaultLockPath, SummaryPath: DefaultSummaryPath}},
		{name: "lock path", config: Config{ConfigPath: DefaultConfigPath, LockPath: " \n", SummaryPath: DefaultSummaryPath}},
		{name: "summary path", config: Config{ConfigPath: DefaultConfigPath, LockPath: DefaultLockPath, SummaryPath: "   "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateFeatureConfig(tt.config); err == nil {
				t.Fatal("ValidateFeatureConfig() error = nil, want non-nil")
			}
		})
	}
}

func TestUpdateSummaryLastKnownGoodRetainsAndMarksSummaryStale(t *testing.T) {
	t.Parallel()

	now := time.Unix(1774433244, 0)
	store := featurekit.NewLastKnownGood[string, puppetcheck.SummarySnapshot](summaryLastKnownGoodStaleAfter)
	summary := puppetcheck.SummarySnapshot{Valid: true, LastRunTimestamp: 1774433000}

	result := updateSummaryLastKnownGood(store, summary, now)
	if !result.Available || result.Stale || result.Value.LastRunTimestamp != summary.LastRunTimestamp {
		t.Fatalf("successful result = %#v, want fresh available summary", result)
	}

	result = updateSummaryLastKnownGood(store, puppetcheck.SummarySnapshot{}, now.Add(time.Hour))
	if !result.Available || result.Stale || result.ConsecutiveFailures != 1 {
		t.Fatalf("failed refresh result = %#v, want fresh retained summary", result)
	}

	result = updateSummaryLastKnownGood(store, puppetcheck.SummarySnapshot{}, now.Add(summaryLastKnownGoodStaleAfter))
	if !result.Available || !result.Stale || result.ConsecutiveFailures != 2 {
		t.Fatalf("stale result = %#v, want stale retained summary", result)
	}
}

func NewFeatureTestSpec() FeatureTestSpec {
	fixtures := filepath.Join("..", "..", "testdata")
	return FeatureTestSpec{
		SuccessfulSnapshot: func(at time.Time) Snapshot {
			summary := puppetcheck.SummarySnapshot{Valid: true}
			return Snapshot{
				puppet: puppetcheck.Snapshot{
					AttemptTime: at,
					Success:     true,
					Config: puppetcheck.ConfigSnapshot{
						Up:    true,
						Valid: true,
					},
					Summary: summary,
				},
				summaryLastKnownGood: successfulSummary(summary, at),
			}
		},
		FailedSnapshot: func(at time.Time, _ error) Snapshot {
			return Snapshot{
				puppet: puppetcheck.Snapshot{
					AttemptTime: at,
					Success:     false,
				},
			}
		},
		ContractFlagArgs: []string{
			"--" + testFeatureName + ".config-path=" + filepath.Join(fixtures, "puppet.conf"),
			"--" + testFeatureName + ".lock-path=" + filepath.Join(fixtures, "agent_disabled.lock"),
			"--" + testFeatureName + ".summary-path=" + filepath.Join(fixtures, "last_run_summary.yaml"),
		},
		ContractRuntimeConfig: map[string]any{
			"config_path":  filepath.Join(fixtures, "puppet.conf"),
			"lock_path":    filepath.Join(fixtures, "agent_disabled.lock"),
			"summary_path": filepath.Join(fixtures, "last_run_summary.yaml"),
		},
		CollectorFlagArgs: []string{
			"--" + testFeatureName + ".config-path=" + filepath.Join(fixtures, "puppet.conf"),
			"--" + testFeatureName + ".lock-path=" + filepath.Join(fixtures, "agent_disabled.lock"),
			"--" + testFeatureName + ".summary-path=" + filepath.Join(fixtures, "last_run_summary.yaml"),
		},
		DefaultRuntimeConfig: map[string]any{
			"config_path":  DefaultConfigPath,
			"lock_path":    DefaultLockPath,
			"summary_path": DefaultSummaryPath,
		},
	}
}

func RegisterFeatureTests(suite *FeatureTestSuite) {
	suite.Register("collector_collects_puppet_metrics", func(t *testing.T) { testCollectorCollectsPuppetMetrics(t, suite) })
	suite.Register("collector_reports_present_lock_without_message_as_disabled", func(t *testing.T) {
		testCollectorReportsPresentLockWithoutMessageAsDisabled(t, suite)
	})
	suite.Register("collector_retains_summary_metrics_after_failed_refresh", func(t *testing.T) {
		testCollectorRetainsSummaryMetricsAfterFailedRefresh(t, suite)
	})
	suite.Register("collector_rejects_duplicate_registration", func(t *testing.T) { testCollectorRejectsDuplicateRegistration(t, suite) })
	suite.Register("exporter_registers_collector_with_fixtures", func(t *testing.T) { testExporterRegistersCollectorWithFixtures(t, suite) })
	suite.Register("exporter_cli_overrides_config_file", func(t *testing.T) { testExporterCLIOverridesConfigFile(t, suite) })
	suite.Register("smoke_spec_includes_puppet_metrics", func(t *testing.T) { testExporterSmokeSpecIncludesPuppetMetrics(t, suite) })
}

func testCollectorReportsPresentLockWithoutMessageAsDisabled(t *testing.T, suite *FeatureTestSuite) {
	now := time.Unix(1774433244, 0)
	collector := suite.NewCollectorWithNow(
		testFeatureName,
		testMetricNamespace,
		nil,
		suite.NewFakeSnapshotter(Snapshot{puppet: puppetcheck.Snapshot{
			AttemptTime: now,
			Success:     true,
			Lock: puppetcheck.LockSnapshot{
				Up:       true,
				Valid:    true,
				Disabled: true,
			},
		}}),
		testRefreshInterval,
		func() time.Time { return now },
	)

	families := exportertest.RegisterAndGather(t, collector)
	exportertest.AssertMetricValue(t, families, "puppet_agent_disabled", map[string]string{"message": ""}, 1)
}

func testCollectorCollectsPuppetMetrics(t *testing.T, suite *FeatureTestSuite) {
	now := time.Unix(1774433244, 0)
	snapshotter := suite.NewFakeSnapshotter(Snapshot{})
	snapshotter.Set(Snapshot{
		puppet: puppetcheck.Snapshot{
			AttemptTime: now,
			Success:     true,
			Config: puppetcheck.ConfigSnapshot{
				Up:                    true,
				Valid:                 true,
				Server:                "puppet.example.net",
				Environment:           "production",
				Certname:              "node-1.example.net",
				MTimeSeconds:          1774433000,
				ScrapeDurationSeconds: 0.01,
			},
			Lock: puppetcheck.LockSnapshot{
				Up:                    true,
				Valid:                 true,
				Disabled:              true,
				DisabledMessage:       "maintenance",
				MTimeSeconds:          1774433010,
				ScrapeDurationSeconds: 0.02,
			},
			Summary: puppetcheck.SummarySnapshot{
				Up:               true,
				Valid:            true,
				Versions:         puppetcheck.Versions{Catalog: "1774217222", Puppet: "8.10.0"},
				RunSuccess:       1,
				LastRunTimestamp: 1774433244,
				Durations:        map[string]float64{"total": 28.5},
				Resources:        map[string]float64{"changed": 2, "total": 42},
				Events:           map[string]float64{"success": 12, "total": 12},
				MTimeSeconds:     1774433020,
			},
		},
		summaryLastKnownGood: successfulSummary(puppetcheck.SummarySnapshot{
			Valid:            true,
			Versions:         puppetcheck.Versions{Catalog: "1774217222", Puppet: "8.10.0"},
			RunSuccess:       1,
			Changes:          2,
			LastRunTimestamp: 1774433244,
			Durations:        map[string]float64{"total": 28.5},
			Resources:        map[string]float64{"changed": 2, "total": 42},
			Events:           map[string]float64{"success": 12, "total": 12},
		}, now),
	})
	collector := suite.NewCollectorWithNow(
		testFeatureName,
		testMetricNamespace,
		nil,
		snapshotter,
		testRefreshInterval,
		func() time.Time { return now },
	)

	families := exportertest.RegisterAndGather(t, collector)
	exportertest.AssertMetricValue(t, families, "puppet_config", map[string]string{
		"puppet_server":      "puppet.example.net",
		"puppet_environment": "production",
		"puppet_certname":    "node-1.example.net",
	}, 1)
	exportertest.AssertMetricValue(t, families, "puppet_config_up", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_config_valid", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_agent_disabled", map[string]string{"message": "maintenance"}, 1)
	exportertest.AssertMetricValue(t, families, "puppet_agent_lock_state_up", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_agent_lock_state_valid", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_success", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_valid", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_timestamp_seconds", nil, 1774433244)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_duration_seconds", map[string]string{"name": "total"}, 28.5)
	exportertest.AssertMetricValue(t, families, "puppet_changes", nil, 2)
	exportertest.AssertMetricValue(t, families, "puppet_resources", map[string]string{"name": "changed"}, 2)
	exportertest.AssertMetricValue(t, families, "puppet_events", map[string]string{"name": "success"}, 12)
	exportertest.AssertMetricValue(t, families, "puppet_version_info", map[string]string{"name": "PuppetVersion", "value": "8.10.0"}, 1)
	exportertest.AssertMetricValue(t, families, "puppet_catalog_version_timestamp_seconds", nil, 1774217222)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_data_available", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_data_stale", nil, 0)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_consecutive_failures", nil, 0)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_last_success_timestamp_seconds", nil, 1774433244)
	exportertest.AssertMetricValue(t, families, testLastSuccess, nil, 1)
	exportertest.AssertMetricValue(t, families, testLastTimestamp, nil, 1774433244)
	exportertest.AssertMetricValue(t, families, testLastSuccessfulTS, nil, 1774433244)
}

func testCollectorRetainsSummaryMetricsAfterFailedRefresh(t *testing.T, suite *FeatureTestSuite) {
	now := time.Unix(1774433244, 0)
	lastSuccess := now.Add(-time.Hour)
	lastKnownGood := successfulSummary(puppetcheck.SummarySnapshot{
		Valid:            true,
		Versions:         puppetcheck.Versions{Catalog: "1774217222", Puppet: "8.10.0"},
		RunSuccess:       1,
		Changes:          3,
		LastRunTimestamp: 1774430000,
	}, lastSuccess)
	lastKnownGood.LastAttempt = now
	lastKnownGood.ConsecutiveFailures = 1
	collector := suite.NewCollectorWithNow(
		testFeatureName,
		testMetricNamespace,
		nil,
		suite.NewFakeSnapshotter(Snapshot{
			puppet: puppetcheck.Snapshot{
				AttemptTime: now,
				Success:     false,
				Config: puppetcheck.ConfigSnapshot{
					ReadErrorsTotal: 1,
				},
				Lock: puppetcheck.LockSnapshot{
					Up:               true,
					ParseErrorsTotal: 1,
				},
				Summary: puppetcheck.SummarySnapshot{
					Up:               true,
					ParseErrorsTotal: 1,
				},
			},
			summaryLastKnownGood: lastKnownGood,
		}),
		testRefreshInterval,
		func() time.Time { return now },
	)

	families := exportertest.RegisterAndGather(t, collector)
	exportertest.AssertMetricValue(t, families, "puppet_config_up", nil, 0)
	exportertest.AssertMetricValue(t, families, "puppet_config_valid", nil, 0)
	exportertest.AssertMetricValue(t, families, "puppet_config_read_errors_total", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_agent_disabled", map[string]string{"message": ""}, 0)
	exportertest.AssertMetricValue(t, families, "puppet_agent_lock_state_up", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_agent_lock_state_valid", nil, 0)
	exportertest.AssertMetricValue(t, families, "puppet_agent_lock_parse_errors_total", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_up", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_valid", nil, 0)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_parse_errors_total", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_data_available", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_consecutive_failures", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_last_run_summary_last_success_timestamp_seconds", nil, float64(lastSuccess.Unix()))
	exportertest.AssertMetricValue(t, families, "puppet_last_run_success", nil, 1)
	exportertest.AssertMetricValue(t, families, "puppet_changes", nil, 3)
	exportertest.AssertMetricValue(t, families, "puppet_catalog_version_timestamp_seconds", nil, 1774217222)
	exportertest.AssertMetricValue(t, families, testLastSuccess, nil, 0)
	assertMetricFamilyAbsent(t, families, "puppet_config")
}

func successfulSummary(summary puppetcheck.SummarySnapshot, at time.Time) featurekit.LastKnownGoodResult[puppetcheck.SummarySnapshot] {
	return featurekit.LastKnownGoodResult[puppetcheck.SummarySnapshot]{
		Value:       summary,
		Available:   true,
		LastSuccess: at,
		LastAttempt: at,
	}
}

func testCollectorRejectsDuplicateRegistration(t *testing.T, suite *FeatureTestSuite) {
	collector := suite.NewCollector(testFeatureName, testMetricNamespace, suite.NewFakeSnapshotter(Snapshot{puppet: puppetcheck.Snapshot{AttemptTime: time.Now(), Success: true}}), testRefreshInterval)
	registry := prometheus.NewRegistry()
	if err := registry.Register(collector); err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}
	if err := registry.Register(collector); err == nil {
		t.Fatal("Register() error = nil, want duplicate collector error")
	}
}

func testExporterRegistersCollectorWithFixtures(t *testing.T, suite *FeatureTestSuite) {
	exporter := suite.NewFeature(featurekit.SpecOptions{
		FeatureName:            testFeatureName,
		DefaultRefreshInterval: 100 * time.Millisecond,
	})
	fixtures := filepath.Join("..", "..", "testdata")
	suite.ParseFeatureFlags(t, exporter, []string{"--" + testFeatureName + ".config-file=" + suite.WriteConfig(t, `
config_path: `+filepath.Join(fixtures, "puppet.conf")+`
lock_path: `+filepath.Join(fixtures, "agent_disabled.lock")+`
summary_path: `+filepath.Join(fixtures, "last_run_summary.yaml")+`
`)})

	registry := suite.RegisterFeatureCollectors(t, exporter)
	exportertest.WaitForMetricValue(t, registry, testLastSuccess, nil, 1)
}

func testExporterCLIOverridesConfigFile(t *testing.T, suite *FeatureTestSuite) {
	exporter := suite.NewFeature(featurekit.SpecOptions{
		FeatureName:            testFeatureName,
		DefaultRefreshInterval: 100 * time.Millisecond,
	})
	fixtures := filepath.Join("..", "..", "testdata")
	suite.ParseFeatureFlags(t, exporter, []string{
		"--" + testFeatureName + ".config-file=" + suite.WriteConfig(t, `
config_path: /does/not/exist/puppet.conf
lock_path: `+filepath.Join(fixtures, "agent_disabled.lock")+`
summary_path: `+filepath.Join(fixtures, "last_run_summary.yaml")+`
`),
		"--" + testFeatureName + ".config-path=" + filepath.Join(fixtures, "puppet.conf"),
	})

	registry := suite.RegisterFeatureCollectors(t, exporter)
	exportertest.WaitForMetricValue(t, registry, testLastSuccess, nil, 1)
}

func testExporterSmokeSpecIncludesPuppetMetrics(t *testing.T, suite *FeatureTestSuite) {
	spec := suite.NewNamedFeature().SmokeSpec()
	for _, want := range []string{
		suite.MetricName(testFeatureName, "", metricConfigUp) + " 1",
		suite.MetricName(testFeatureName, "", metricAgentLockStateUp) + " 1",
		suite.MetricName(testFeatureName, "", metricCatalogVersionTimestamp) + " 1.774227222e+09",
		suite.MetricName(testFeatureName, "", metricLastRunSummaryUp) + " 1",
	} {
		if !featuretest.HasString(spec.WantMetrics, want) {
			t.Fatalf("SmokeSpec().WantMetrics = %v, want %q", spec.WantMetrics, want)
		}
	}
	for _, want := range []string{
		"--" + testFeatureName + ".config-file=../examples/" + DefaultFeatureConfigFileName,
		"--" + testFeatureName + ".config-path=../testdata/puppet.conf",
		"--" + testFeatureName + ".lock-path=../testdata/agent_disabled.lock",
		"--" + testFeatureName + ".summary-path=../testdata/last_run_summary.yaml",
	} {
		if !featuretest.HasString(spec.ServerArgs, want) {
			t.Fatalf("SmokeSpec().ServerArgs = %v, want %q", spec.ServerArgs, want)
		}
	}
}

func assertMetricFamilyAbsent(t *testing.T, families []*dto.MetricFamily, name string) {
	t.Helper()
	for _, family := range families {
		if family.GetName() == name {
			t.Fatalf("unexpected metric family %q", name)
		}
	}
}
