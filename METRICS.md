# Metrics

Puppet domain metrics use the `puppet` feature namespace. Framework-owned
exporter metrics use the `puppet_exporter` metric namespace.

## Source Health

The exporter reads three local Puppet agent sources:

- `puppet.conf`
- `agent_disabled.lock`
- `last_run_summary.yaml`

Current source state is represented by `up` and `valid` gauges:

- `up = 1` means the source was readable during the last collection.
- `up = 0` means the source could not be read.
- `valid = 1` means the readable source was parsed and interpreted
  successfully.
- `valid = 0` with `up = 1` means the source is readable but invalid.

Missing `agent_disabled.lock` is treated as a healthy enabled-agent state, so
`puppet_agent_lock_state_up = 1`, `puppet_agent_lock_state_valid = 1`, and
`puppet_agent_disabled{message=""} = 0`.
The presence of a valid lock file means the agent is disabled even when its
optional `disabled_message` field is absent or empty.

### Current State Gauges

`puppet_config_up`

Whether `puppet.conf` was readable during the last collection.

`puppet_config_valid`

Whether `puppet.conf` contains non-empty `server`, `environment`, and
`certname` values.

`puppet_agent_lock_state_up`

Whether `agent_disabled.lock` was readable, or absent because the agent is
enabled.

`puppet_agent_lock_state_valid`

Whether `agent_disabled.lock` was absent or parsed successfully.

`puppet_last_run_summary_up`

Whether `last_run_summary.yaml` was readable during the last collection.

`puppet_last_run_summary_valid`

Whether `last_run_summary.yaml` was parsed and interpreted successfully. A
valid summary must contain `events.failure`, `resources.failed`, and
`time.last_run`; a readable summary missing any of these fields is invalid.
Exported counters and durations must be finite and non-negative.
`time.last_run` must be positive and no more than five minutes in the future.
A numeric `version.config` catalog timestamp follows the same future-skew
limit; non-numeric custom catalog versions remain valid but do not produce
`puppet_catalog_version_timestamp_seconds`.

### Last-Known-Good Summary State

The exporter retains the most recent valid `last_run_summary.yaml` snapshot
when a later read or parse attempt fails. Current `*_up` and `*_valid` gauges
still describe the latest attempt, while summary-derived business metrics use
the retained value until a newer valid summary is collected.

`puppet_last_run_summary_last_success_timestamp_seconds`

Unix timestamp of the most recent valid summary collection.

`puppet_last_run_summary_consecutive_failures`

Number of consecutive failed summary collection attempts since the last
success.

`puppet_last_run_summary_data_available`

Whether a valid current or retained summary is available.

`puppet_last_run_summary_data_stale`

Whether retained summary data is available but no valid summary has been
collected for at least two hours.

### Source Freshness and Duration

`puppet_config_mtime_seconds`

Unix mtime of `puppet.conf`, or `0` if unavailable.

`puppet_agent_lock_mtime_seconds`

Unix mtime of `agent_disabled.lock`, or `0` when the lock file is absent or
unavailable.

`puppet_last_run_summary_mtime_seconds`

Unix mtime of `last_run_summary.yaml`, or `0` if unavailable.

`puppet_config_scrape_duration_seconds`

Duration of the latest `puppet.conf` refresh.

`puppet_agent_lock_scrape_duration_seconds`

Duration of the latest `agent_disabled.lock` refresh.

`puppet_last_run_summary_scrape_duration_seconds`

Duration of the latest `last_run_summary.yaml` refresh.

### Source Error Counters

`puppet_config_read_errors_total`

`puppet_agent_lock_read_errors_total`

`puppet_last_run_summary_read_errors_total`

Cumulative source read/open error counters.

`puppet_config_parse_errors_total`

`puppet_agent_lock_parse_errors_total`

`puppet_last_run_summary_parse_errors_total`

Cumulative source parse/decode error counters.

These are counters. Use `rate()` or `increase()` for history; do not use them
as current failure counts.

## Puppet State

`puppet_config`

Puppet configuration labels. The metric value is always `1`.

Labels:

- `puppet_server`
- `puppet_environment`
- `puppet_certname`

Emitted only when `puppet_config_valid = 1`.

`puppet_agent_disabled`

Whether the Puppet agent is disabled.

Labels:

- `message`

The metric is emitted as `1` whenever a valid lock file is present. The
`message` label is informational and can be empty. The metric is emitted as `0`
with an empty message when the lock source is missing or invalid.

`puppet_version_info`

Puppet version metadata. The metric value is always `1`.

Labels:

- `name`
- `value`

Current exported entries include `CatalogVersion` and `PuppetVersion`.

`puppet_catalog_version_timestamp_seconds`

Unix timestamp encoded in Puppet's `version.config` catalog version. The metric
is emitted only when the catalog version is a positive numeric value. It can be
used to alert when an agent has not applied a new catalog recently. During a
transient summary failure, the metric retains its last-known-good value.

`puppet_last_run_success`

Whether the last Puppet run succeeded. The exporter derives this from
`events.failure == 0` and `resources.failed == 0`.

`puppet_last_run_timestamp_seconds`

Unix timestamp from `time.last_run`.

`puppet_last_run_duration_seconds`

Duration values from `time.*`, excluding `last_run`.

Labels:

- `name`

`puppet_resources`

Resource counters from `resources.*`.

Labels:

- `name`

Important names used by the dashboard and alert rules:

- `failed`
- `failed_to_restart`
- `out_of_sync`
- `total`

`failed`, `failed_to_restart`, and `out_of_sync` are treated as critical when
greater than `0`. Other non-`total` resource counters are informational or
warning-oriented, depending on the dashboard panel. `total` is informational.

`puppet_events`

Event counters from `events.*`.

Labels:

- `name`

Important names used by the dashboard and alert rules:

- `failure`
- `success`
- `total`

`failure` is treated as critical when greater than `0`. `success` and `total`
are informational.

## Exporter Collection

`puppet_exporter_last_collection_success`

Whether the latest background Puppet refresh succeeded.

`puppet_exporter_last_collection_timestamp_seconds`

Unix timestamp of the latest background Puppet refresh attempt. The value is
`0` before the first attempt.

`puppet_exporter_last_successful_collection_timestamp_seconds`

Unix timestamp of the latest successful background Puppet refresh. The value is
`0` until the first successful refresh.

`puppet_exporter_collection_duration_seconds`

Histogram of framework collection refresh duration. It is emitted by the
framework collector and uses the standard Prometheus histogram series:

- `puppet_exporter_collection_duration_seconds_bucket`
- `puppet_exporter_collection_duration_seconds_sum`
- `puppet_exporter_collection_duration_seconds_count`

## Build Info

`puppet_exporter_build_info`

Build and runtime metadata exposed by the framework. Labels include version,
revision, branch, Go version, GOOS, and GOARCH. The metric value is always `1`.
The Grafana inventory table displays only the `version` label from this metric.
