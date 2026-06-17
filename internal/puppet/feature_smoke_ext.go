package puppet

import "github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter/featurekit"

func FeatureSmoke(ctx featurekit.SmokeContext[Config]) featurekit.SmokeSpec {
	return featurekit.SmokeSpec{
		ServerArgs: []string{
			"--" + ctx.FeatureName + ".config-file=../examples/" + DefaultFeatureConfigFileName,
			"--" + ctx.FeatureName + ".config-path=../testdata/puppet.conf",
			"--" + ctx.FeatureName + ".lock-path=../testdata/agent_disabled.lock",
			"--" + ctx.FeatureName + ".summary-path=../testdata/last_run_summary.yaml",
		},
		WantMetrics: []string{
			featurekit.FeatureMetricName(ctx.FeatureName, "", metricConfigUp, featureMetricSpecs) + " 1",
			featurekit.FeatureMetricName(ctx.FeatureName, "", metricAgentLockStateUp, featureMetricSpecs) + " 1",
			featurekit.FeatureMetricName(ctx.FeatureName, "", metricAgentLockStateValid, featureMetricSpecs) + " 1",
			featurekit.FeatureMetricName(ctx.FeatureName, "", metricLastRunSummaryUp, featureMetricSpecs) + " 1",
			featurekit.FeatureMetricName(ctx.FeatureName, "", metricLastRunSummaryValid, featureMetricSpecs) + " 1",
		},
		RejectMetrics: []string{
			featurekit.FeatureMetricName(ctx.FeatureName, "", metricConfigUp, featureMetricSpecs) + " 0",
			featurekit.FeatureMetricName(ctx.FeatureName, "", metricAgentLockStateUp, featureMetricSpecs) + " 0",
			featurekit.FeatureMetricName(ctx.FeatureName, "", metricLastRunSummaryUp, featureMetricSpecs) + " 0",
		},
	}
}
