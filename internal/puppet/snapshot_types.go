package puppet

import "github.com/zxzharmlesszxz/prometheus-puppet-exporter/internal/puppetcheck"

type Snapshot struct {
	puppet puppetcheck.Snapshot
}
