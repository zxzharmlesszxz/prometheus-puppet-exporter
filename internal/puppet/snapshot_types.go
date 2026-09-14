package puppet

import (
	"github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter/featurekit"
	"github.com/zxzharmlesszxz/prometheus-puppet-exporter/internal/puppetcheck"
)

type Snapshot struct {
	puppet               puppetcheck.Snapshot
	summaryLastKnownGood featurekit.LastKnownGoodResult[puppetcheck.SummarySnapshot]
}
