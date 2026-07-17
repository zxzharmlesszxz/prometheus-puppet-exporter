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

func NewFeatureTestSpec() FeatureTestSpec {
	fixtures := filepath.Join("..", "..", "testdata")
	return FeatureTestSpec{
		SuccessfulSnapshot: func(at time.Time) Snapshot {
			return Snapshot{
				puppet: puppetcheck.Snapshot{
					AttemptTime: at,
					Success:     true,
					Config: puppetcheck.ConfigSnapshot{
						Up:    true,
						Valid: true,
					},
				},
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
	suite.Register("collector_omits_business_metrics_for_failed_sources", func(t *testing.T) { testCollectorOmitsBusinessMetricsForFailedSources(t, suite) })
	suite.Register("collector_rejects_duplicate_registration", func(t *testing.T) { testCollectorRejectsDuplicateRegistration(t, suite) })
	suite.Register("exporter_registers_collector_with_fixtures", func(t *testing.T) { testExporterRegistersCollectorWithFixtures(t, suite) })
	suite.Register("exporter_cli_overrides_config_file", func(t *testing.T) { testExporterCLIOverridesConfigFile(t, suite) })
	suite.Register("smoke_spec_includes_puppet_metrics", func(t *testing.T) { testExporterSmokeSpecIncludesPuppetMetrics(t, suite) })
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
	exportertest.AssertMetricValue(t, families, "puppet_resources", map[string]string{"name": "changed"}, 2)
	exportertest.AssertMetricValue(t, families, "puppet_events", map[string]string{"name": "success"}, 12)
	exportertest.AssertMetricValue(t, families, "puppet_version_info", map[string]string{"name": "PuppetVersion", "value": "8.10.0"}, 1)
	exportertest.AssertMetricValue(t, families, "puppet_catalog_version_timestamp_seconds", nil, 1774217222)
	exportertest.AssertMetricValue(t, families, testLastSuccess, nil, 1)
	exportertest.AssertMetricValue(t, families, testLastTimestamp, nil, 1774433244)
	exportertest.AssertMetricValue(t, families, testLastSuccessfulTS, nil, 1774433244)
}

func testCollectorOmitsBusinessMetricsForFailedSources(t *testing.T, suite *FeatureTestSuite) {
	now := time.Unix(1774433244, 0)
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
	exportertest.AssertMetricValue(t, families, testLastSuccess, nil, 0)
	assertMetricFamilyAbsent(t, families, "puppet_config")
	assertMetricFamilyAbsent(t, families, "puppet_last_run_success")
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
