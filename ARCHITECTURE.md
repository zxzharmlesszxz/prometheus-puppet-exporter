# Architecture

`prometheus-puppet-exporter` is a thin concrete exporter built on `prometheus-exporter-framework`.

## Package Layout

- `cmd`
  Minimal process entrypoint. The generated entrypoint file is
  `scaffold_main.go` and should stay scaffold-owned.
- `internal/exporter`
  Thin adapter that asks the feature package for a contract-backed feature and
  delegates bootstrap metadata to the framework. Files named `scaffold_*.go`
  are fully scaffold-owned.
- `internal/puppet`
  Concrete feature package. `scaffold_feature.go` owns the scaffold-compatible
  `featurekit.SnapshotFeatureExtension` assembly and wires config-file flags,
  runtime config, collector construction, metrics, snapshot status, and smoke
  behavior through Puppet-specific hooks.
  `snapshot_types.go` is the scaffold-managed `Snapshot` struct that wraps
  `internal/puppetcheck.Snapshot`, but it is feature-owned after generation
  so a concrete exporter can replace it with an aggregate snapshot.
  Feature-specific defaults and hook functions live in adjacent feature files:
  `feature_config_ext.go`, `feature_metrics_ext.go`,
  `feature_snapshotter_ext.go`, and `feature_smoke_ext.go`.
  `scaffold_feature_test_suite_test.go` owns the thin scaffold bridge into
  framework `exporter/exportertest/featuretest`. Register feature-specific
  test cases from `feature_test_suite_ext_test.go` instead of editing
  scaffold-owned tests.
- `internal/puppetcheck`
  Puppet file-check engine: `puppet.conf`, `agent_disabled.lock`, and
  `last_run_summary.yaml` parsing, counters, and composite snapshot types.
- `examples/grafana`
  Example Grafana dashboard for fleet overview and per-instance inspection via
  dashboard variables.
- `examples/prometheus`
  Example Prometheus alert rules for exporter availability, source failures,
  stale data, failed runs, resource/event failures, and disabled agents.

Concrete exporter logic belongs in non-`scaffold_*.go` files and domain check
packages. Treat `scaffold_*.go` files as generated contract glue and update
them through the scaffold sync flow only.

## Data Flow

1. `cmd/scaffold_main.go` delegates to `internal/exporter.Main()`, which runs `framework.MainFromInjectedProject(...)`.
2. `internal/exporter` creates the concrete feature through
   `internal/puppet.NewFeature(...)` and framework-injected feature metadata.
3. Framework `featurekit.Feature` registers common flags such as `--puppet.refresh-interval` and `--puppet.config-file`, then delegates feature-specific flag specs and behavior through the framework-owned feature contract.
4. Framework `featurekit.Feature` builds a typed snapshotter and collector from the extension-backed spec, then registers and starts the collector.
5. `framework.SnapshotCollector` refreshes data in a background worker every `--puppet.refresh-interval`; scrapes read the latest completed snapshot.
6. The feature snapshotter delegates backing-file reads to `internal/puppetcheck`.
7. A single background snapshot collector refreshes all three backing files.
8. Prometheus scrapes expose the latest successful or failed composite snapshot.
9. If a source cannot be read, the collector emits that source's `*_up`
   metric as `0`.
10. If a source is readable but invalid, the collector emits that source's
    `*_up` metric as `1` and its `*_valid` metric as `0`.
11. Config and summary business metrics are omitted when their source is
    invalid.
12. Lock state still emits `puppet_agent_disabled{message=""} 0` on source
    errors so the lock-state metric remains present, while
    `puppet_agent_lock_state_up` and `puppet_agent_lock_state_valid` expose the
    failure mode.

## Failure Semantics

If refresh fails before the first successful snapshot, the exporter exposes collection health metrics, but no business metrics.

If the latest refresh fails, the exporter exposes collection health metrics, but no business metrics, and sets:

- `puppet_exporter_last_collection_success = 0`

The `/healthz` endpoint remains `200 OK` while the process is alive even if the latest collection failed.

Read and parse error metrics are cumulative counters. Dashboards and alerts use
`*_up` and `*_valid` for current failure state, and use the counters only with
`rate()` or `increase()` for historical diagnostics.

## Dashboard Shape

The bundled Grafana dashboards use the Grafana v2 dashboard resource model. The
main dashboard is organized into:

- `Overview`: `Status`, `Main Metrics`, collapsed `Source Health`, collapsed
  `Historical Graph`, and collapsed `Exporter Collection`.
- `Runtime`: framework Go/process runtime panels.
- `Scrape`: Prometheus-side scrape health panels.

`Main Metrics` contains current Puppet state stats, `Resources` and `Events`
stat panels, a full-width `Puppet Inventory` table, and three Puppet state
graphs in one row. Tables disable pagination and rely on scrolling.

`Puppet Inventory` is the primary fleet table. It joins these series by
`instance`:

- `puppet_config`
- `puppet_exporter_build_info`
- `puppet_version_info{name="PuppetVersion"}`
- `puppet_resources{name="failed"}`
- `puppet_last_run_timestamp_seconds`
- `puppet_last_run_success`
- `puppet_agent_disabled`
- `puppet_config_valid`
- `puppet_agent_lock_state_valid`
- `puppet_last_run_summary_valid`

The visible columns are `instance`, exporter version, certname, environment,
Puppet server, Puppet version, failed resources, last-run age, run status,
disabled state, source validity, and disabled-agent message.

## Alert Shape

The bundled Prometheus rules cover:

- exporter scrape availability
- framework background collection failure
- source read failures and readable-but-invalid source states
- stale `last_run_summary.yaml` mtime and stale Puppet run timestamps
- failed Puppet runs
- critical resource states: `failed`, `failed_to_restart`, and `out_of_sync`
- event failures via `puppet_events{name="failure"}`
- disabled Puppet agents
- read/parse counter increases over a 30 minute window

## Testing Strategy

The project uses:

- unit tests for snapshot parsing, metric emission, and feature wiring
- scaffold-owned handler tests for HTTP integration
- fixture-backed smoke tests using `testdata/`

`docker-compose.yml` also uses the same local fixtures so local manual testing matches the repository test inputs closely.
The Compose stack also provisions the example Grafana dashboards from `examples/grafana/`.
[`METRICS.md`](METRICS.md) contains the canonical metric contract.
