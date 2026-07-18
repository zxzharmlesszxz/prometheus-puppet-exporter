# prometheus-puppet-exporter

Small Prometheus exporter that reads local Puppet agent state and exposes it as metrics.

It does not talk to Puppet Server, PuppetDB, or external APIs. The exporter is intentionally file-backed: it reports only what is present on disk in the local Puppet agent files.
The exporter also exposes source-health metrics for each input file, so Prometheus can distinguish readable, parseable state from missing or malformed inputs.

## Inputs

The exporter reads three local Puppet agent files:

- `puppet.conf`
- `last_run_summary.yaml`
- `agent_disabled.lock`

## Local Run

```bash
make build
./dist/prometheus-puppet-exporter \
  --web.listen-address=:9819 \
  --puppet.config-file=examples/prometheus-puppet-exporter.yml
```

Useful flags:

```bash
--puppet.config-file          # Path to the YAML feature config file
--puppet.config-path          # Path to puppet.conf
--puppet.lock-path            # Path to agent_disabled.lock
--puppet.summary-path         # Path to last_run_summary.yaml
--puppet.refresh-interval     # Background refresh interval (default: 1m)
--web.listen-address          # HTTP listen address (default: :9819)
--web.telemetry-path          # Metrics path (default: /metrics)
--web.enable-pprof            # Enable pprof endpoints
--log.level                   # Log level (default: info)
--log.format                  # Log format (default: logfmt)
```

By default, the exporter listens on `:9819` and refreshes data every `1m`.
If no `--puppet.config-file` value is provided, `/etc/prometheus/prometheus-puppet-exporter.yml` is loaded when it exists; if it is missing, defaults and flags are used.
The shipped `examples/prometheus-puppet-exporter.yml` file configures the default Puppet file paths.
Make, Compose, and smoke defaults use `FEATURE_CONFIG_FILE=prometheus-puppet-exporter.yml` and pass that path explicitly with `--puppet.config-file=...`.
Runtime config can always be overridden with another `--puppet.config-file=...` value.
Data refresh runs through the framework snapshot collector in a background worker; scrapes return the last collected snapshot.

## Configuration Example

The YAML config file accepts these Puppet-specific keys:

```yaml
config_path: /etc/puppetlabs/puppet/puppet.conf
lock_path: /opt/puppetlabs/puppet/public/agent_disabled.lock
summary_path: /opt/puppetlabs/puppet/public/last_run_summary.yaml
```

When the exporter runs in a container, mount the host Puppet state and point the
paths at the mounted tree:

```yaml
config_path: /host/etc/puppetlabs/puppet/puppet.conf
lock_path: /host/opt/puppetlabs/puppet/public/agent_disabled.lock
summary_path: /host/opt/puppetlabs/puppet/public/last_run_summary.yaml
```

The runtime image runs as the non-root `nobody` user. Every parent directory
must be searchable and each configured Puppet file must be readable by the
container runtime UID/GID. Prefer granting group read access or configuring an
explicit runtime user through Compose `user` or a Kubernetes
`securityContext`; do not make Puppet state writable by the exporter.

## Metrics

Example output:

```code
puppet_config{puppet_server="puppet.example.net",puppet_environment="production",puppet_certname="node-1.example.net"} 1
puppet_config_up 1
puppet_config_valid 1
puppet_agent_lock_state_up 1
puppet_agent_lock_state_valid 1
puppet_last_run_summary_up 1
puppet_last_run_summary_valid 1
puppet_last_run_success 1
puppet_last_run_timestamp_seconds 1774217244
puppet_last_run_duration_seconds{name="total"} 28.023470087
puppet_agent_disabled{message=""} 0
puppet_exporter_last_collection_success 1
```

Important metric groups:

- `puppet_config`
- `puppet_config_up`
- `puppet_config_valid`
- `puppet_version_info`
- `puppet_catalog_version_timestamp_seconds`
- `puppet_last_run_success`
- `puppet_last_run_summary_up`
- `puppet_last_run_timestamp_seconds`
- `puppet_last_run_duration_seconds`
- `puppet_resources`
- `puppet_events`
- `puppet_agent_disabled`
- `puppet_agent_lock_state_up`
- `puppet_agent_lock_state_valid`
- `puppet_last_run_summary_valid`
- `puppet_exporter_last_collection_success`
- `puppet_exporter_collection_duration_seconds`
- `puppet_exporter_build_info`

The full metric contract lives in [`METRICS.md`](METRICS.md).

## Input Semantics

Each source follows a simple health convention:

- `*_up = 1` means the source was readable during the last collection
- `*_up = 0` means the source could not be read
- `*_valid = 1` means the readable source was parsed and interpreted
  successfully
- `*_valid = 0` with `*_up = 1` means the source is readable but invalid

Read and parse error metrics such as `puppet_config_read_errors_total` are
cumulative counters. Use `rate()` or `increase()` for history; use `*_up` and
`*_valid` for current state.

The exporter derives `puppet_last_run_success` locally from summary counters instead of exposing it as a raw Puppet field.
Missing `agent_disabled.lock` is treated as a valid enabled-agent state.
A present, valid lock file means the agent is disabled even when its optional
`disabled_message` field is empty or absent.
Required Puppet config values and configured source paths must contain
non-whitespace characters. Summary counters and durations must be finite and
non-negative; last-run and numeric catalog timestamps allow at most five
minutes of future clock skew.

## Docker Compose

The repository includes [`docker-compose.yml`](docker-compose.yml) for local testing.
The Prometheus scrape config is embedded in Compose, while alerting rules live
under [`examples/prometheus`](examples/prometheus).
It starts:

- `exporter`
- `prometheus`
- `grafana`

```bash
make compose
```

Endpoints:

- `http://localhost:9819`
- `http://localhost:9819/metrics`
- `http://localhost:9819/healthz`
- `http://localhost:9090`
- `http://localhost:3000`

## Grafana

Docker Compose provisions Grafana with:

- Prometheus datasource `DS_PROMETHEUS`
- dashboards from [`examples/grafana`](examples/grafana)
- default login `admin` / `admin`

Open `http://localhost:3000` after `make compose`.

For a direct Docker build, run:

```bash
make docker-build
```

## Tests

```bash
make go-check
```

The repository includes the same maintenance target layout used by the concrete exporter repos:

```bash
make help
make go-check
make check
make docker-smoke
make full-check
```

`make go-check` runs Go-only checks. `make check` also validates the Prometheus and Docker Compose examples, so it requires Docker.

## Alert Rules

Example Prometheus alert rules live in [`examples/prometheus/prometheus-puppet-exporter.yml`](examples/prometheus/prometheus-puppet-exporter.yml).
They cover exporter scrape availability, framework collection failure,
source read/validity failures, stale Puppet data, failed Puppet runs, resource
problems, event failures, disabled agents, and read/parse counter spikes.
The bundled exporter-down rule and Grafana job variable expect the scrape job
name to match `^(prometheus[_-])?puppet[_-]exporter$`; adapt that matcher when
using another job naming convention.

## Grafana Dashboards

The example Grafana dashboard lives in
[`examples/grafana/prometheus-puppet-exporter.json`](examples/grafana/prometheus-puppet-exporter.json).
It uses the Grafana v2 dashboard resource model with `Overview`, `Runtime`, and
`Scrape` tabs.

The `Overview` tab contains:

- `Status`: exporter collection status and age.
- `Main Metrics`: current Puppet status stats, `Resources` and `Events` stat
  panels, a full-width `Puppet Inventory` table, and Puppet state graphs.
- collapsed `Source Health`, `Historical Graph`, and `Exporter Collection`
  rows.

`Puppet Inventory` joins Puppet config, exporter build version, Puppet version,
failed resources, last-run age, run success, disabled state, source validity,
and disabled-agent message by `instance`.

## Scaffold-Owned Go Files

Go files named `scaffold_*.go` are generated contract glue and should stay
identical to the scaffold output. Add Puppet-specific behavior in adjacent
non-scaffold files such as `internal/puppet/*_ext.go`, `internal/puppet/metrics.go`,
and the `internal/puppetcheck` domain package.
The generated starter `Snapshot` struct lives in `snapshot_types.go`, and the
starter snapshot structure lives in `internal/puppetcheck`; both are
feature-owned after generation so concrete exporters can replace the struct
with an aggregate snapshot and split domain checks into multiple packages.
Standard feature tests run through framework `exporter/exportertest/featuretest`
via `scaffold_feature_test_suite_test.go`; add feature-specific test cases by
registering them in `feature_test_suite_ext_test.go`.

Build local release artifacts:

```bash
make build VERSION=v0.1.0
make release VERSION=v0.1.0
make release-smoke VERSION=v0.1.0
```

Build and push a Docker image:

```bash
make docker-build VERSION=v0.1.0 DOCKER_IMAGE=prometheus-puppet-exporter:v0.1.0
make docker-push DOCKER_IMAGE=prometheus-puppet-exporter:v0.1.0
make docker-buildx-push VERSION=v0.1.0 DOCKER_IMAGE=registry.example.com/prometheus-puppet-exporter:v0.1.0
```

## Architecture

The high-level design is documented in [`ARCHITECTURE.md`](ARCHITECTURE.md).

## License

This project is licensed under the MIT License. See [`LICENSE`](LICENSE).
