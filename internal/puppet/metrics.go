package puppet

import "github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter/featurekit"

const (
	metricConfig                       = "config"
	metricConfigUp                     = "config_up"
	metricConfigValid                  = "config_valid"
	metricConfigMTime                  = "config_mtime"
	metricConfigReadErrors             = "config_read_errors"
	metricConfigParseErrors            = "config_parse_errors"
	metricConfigScrapeDuration         = "config_scrape_duration"
	metricAgentDisabled                = "agent_disabled"
	metricAgentLockStateUp             = "agent_lock_state_up"
	metricAgentLockStateValid          = "agent_lock_state_valid"
	metricAgentLockMTime               = "agent_lock_mtime"
	metricAgentLockReadErrors          = "agent_lock_read_errors"
	metricAgentLockParseErrors         = "agent_lock_parse_errors"
	metricAgentLockScrapeDuration      = "agent_lock_scrape_duration"
	metricVersionInfo                  = "version_info"
	metricLastRunDuration              = "last_run_duration"
	metricLastRunTimestamp             = "last_run_timestamp"
	metricResources                    = "resources"
	metricEvents                       = "events"
	metricLastRunSuccess               = "last_run_success"
	metricLastRunSummaryUp             = "last_run_summary_up"
	metricLastRunSummaryValid          = "last_run_summary_valid"
	metricLastRunSummaryMTime          = "last_run_summary_mtime"
	metricLastRunSummaryReadErrors     = "last_run_summary_read_errors"
	metricLastRunSummaryParseErrors    = "last_run_summary_parse_errors"
	metricLastRunSummaryScrapeDuration = "last_run_summary_scrape_duration"
)

var featureMetricSpecs = []featurekit.FeatureMetricSpec{
	{
		ID:     metricConfig,
		Scope:  featurekit.MetricScopeFeature,
		Name:   "_config",
		Help:   "Puppet configuration.",
		Labels: []string{"puppet_server", "puppet_environment", "puppet_certname"},
	},
	{
		ID:    metricConfigUp,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_config_up",
		Help:  "1 if puppet.conf was readable during the last collection.",
	},
	{
		ID:    metricConfigValid,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_config_valid",
		Help:  "1 if puppet.conf contains non-empty required fields.",
	},
	{
		ID:    metricConfigMTime,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_config_mtime_seconds",
		Help:  "Unix timestamp of the puppet.conf file modification time, or 0 if unavailable.",
	},
	{
		ID:    metricConfigReadErrors,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_config_read_errors_total",
		Help:  "Total number of puppet.conf read errors.",
	},
	{
		ID:    metricConfigParseErrors,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_config_parse_errors_total",
		Help:  "Total number of puppet.conf parse errors.",
	},
	{
		ID:    metricConfigScrapeDuration,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_config_scrape_duration_seconds",
		Help:  "Time spent refreshing metrics from puppet.conf.",
	},
	{
		ID:     metricAgentDisabled,
		Scope:  featurekit.MetricScopeFeature,
		Name:   "_agent_disabled",
		Help:   "1 if the Puppet agent is disabled via agent_disabled.lock.",
		Labels: []string{"message"},
	},
	{
		ID:    metricAgentLockStateUp,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_agent_lock_state_up",
		Help:  "1 if agent_disabled.lock was readable, or absent because the Puppet agent is enabled.",
	},
	{
		ID:    metricAgentLockStateValid,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_agent_lock_state_valid",
		Help:  "1 if agent_disabled.lock was absent or parsed successfully.",
	},
	{
		ID:    metricAgentLockMTime,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_agent_lock_mtime_seconds",
		Help:  "Unix timestamp of the agent_disabled.lock file modification time, or 0 if unavailable.",
	},
	{
		ID:    metricAgentLockReadErrors,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_agent_lock_read_errors_total",
		Help:  "Total number of agent_disabled.lock read errors.",
	},
	{
		ID:    metricAgentLockParseErrors,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_agent_lock_parse_errors_total",
		Help:  "Total number of agent_disabled.lock parse errors.",
	},
	{
		ID:    metricAgentLockScrapeDuration,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_agent_lock_scrape_duration_seconds",
		Help:  "Time spent refreshing metrics from agent_disabled.lock.",
	},
	{
		ID:     metricVersionInfo,
		Scope:  featurekit.MetricScopeFeature,
		Name:   "_version_info",
		Help:   "The version of Puppet agent.",
		Labels: []string{"value", "name"},
	},
	{
		ID:     metricLastRunDuration,
		Scope:  featurekit.MetricScopeFeature,
		Name:   "_last_run_duration_seconds",
		Help:   "Duration of the last Puppet run.",
		Labels: []string{"name"},
	},
	{
		ID:    metricLastRunTimestamp,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_last_run_timestamp_seconds",
		Help:  "Unix timestamp of the last Puppet run.",
	},
	{
		ID:     metricResources,
		Scope:  featurekit.MetricScopeFeature,
		Name:   "_resources",
		Help:   "Puppet resources.",
		Labels: []string{"name"},
	},
	{
		ID:     metricEvents,
		Scope:  featurekit.MetricScopeFeature,
		Name:   "_events",
		Help:   "Puppet events.",
		Labels: []string{"name"},
	},
	{
		ID:    metricLastRunSuccess,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_last_run_success",
		Help:  "1 if the last Puppet run was successful.",
	},
	{
		ID:    metricLastRunSummaryUp,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_last_run_summary_up",
		Help:  "1 if the last_run_summary.yaml file was readable.",
	},
	{
		ID:    metricLastRunSummaryValid,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_last_run_summary_valid",
		Help:  "1 if last_run_summary.yaml was parsed and interpreted successfully.",
	},
	{
		ID:    metricLastRunSummaryMTime,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_last_run_summary_mtime_seconds",
		Help:  "Unix timestamp of the last_run_summary.yaml file modification time, or 0 if unavailable.",
	},
	{
		ID:    metricLastRunSummaryReadErrors,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_last_run_summary_read_errors_total",
		Help:  "Total number of last_run_summary.yaml read errors.",
	},
	{
		ID:    metricLastRunSummaryParseErrors,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_last_run_summary_parse_errors_total",
		Help:  "Total number of last_run_summary.yaml parse errors.",
	},
	{
		ID:    metricLastRunSummaryScrapeDuration,
		Scope: featurekit.MetricScopeFeature,
		Name:  "_last_run_summary_scrape_duration_seconds",
		Help:  "Time spent refreshing metrics from last_run_summary.yaml.",
	},
}
