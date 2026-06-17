package puppet

import (
	"context"
	"time"

	framework "github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter"
	"github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter/featurekit"
	"github.com/zxzharmlesszxz/prometheus-puppet-exporter/internal/puppetcheck"
)

func NewDefaultSnapshotEngine() featurekit.SnapshotEngine[Snapshot] {
	engine, err := newSnapshotEngine(NewDefaultConfig())
	if err != nil {
		panic(err)
	}
	return engine
}

func NewSnapshotEngine(ctx featurekit.CollectorContext[Config]) (featurekit.SnapshotEngine[Snapshot], error) {
	config, _, _, err := ResolveFeatureConfig(ctx.FeatureName, ctx.Config)
	if err != nil {
		return nil, err
	}
	return newSnapshotEngine(config)
}

func FeatureSnapshotStatus(snapshot Snapshot) framework.SnapshotStatus {
	return framework.SnapshotStatus{
		AttemptTime: snapshot.puppet.AttemptTime,
		Success:     snapshot.puppet.Success,
	}
}

func newSnapshotEngine(config Config) (featurekit.SnapshotEngine[Snapshot], error) {
	checker := puppetcheck.NewChecker(puppetcheck.Paths{
		Config:  config.ConfigPath,
		Lock:    config.LockPath,
		Summary: config.SummaryPath,
	})
	return featurekit.SnapshotEngineFunc[Snapshot](func(ctx context.Context, now time.Time) Snapshot {
		return Snapshot{
			puppet: checker.Snapshot(ctx, now),
		}
	}), nil
}
