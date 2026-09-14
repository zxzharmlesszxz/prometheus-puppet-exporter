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
	return newSnapshotEngine(ctx.Config)
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
	summaryLastKnownGood := featurekit.NewLastKnownGood[string, puppetcheck.SummarySnapshot](summaryLastKnownGoodStaleAfter)
	return featurekit.SnapshotEngineFunc[Snapshot](func(ctx context.Context, now time.Time) Snapshot {
		puppetSnapshot := checker.Snapshot(ctx, now)
		return Snapshot{
			puppet:               puppetSnapshot,
			summaryLastKnownGood: updateSummaryLastKnownGood(summaryLastKnownGood, puppetSnapshot.Summary, now),
		}
	}), nil
}

func updateSummaryLastKnownGood(store *featurekit.LastKnownGood[string, puppetcheck.SummarySnapshot], summary puppetcheck.SummarySnapshot, now time.Time) featurekit.LastKnownGoodResult[puppetcheck.SummarySnapshot] {
	if summary.Valid {
		store.Success(lastRunSummarySource, summary, now)
	} else {
		store.Failure(lastRunSummarySource, now)
	}
	return store.Get(lastRunSummarySource, now)
}
