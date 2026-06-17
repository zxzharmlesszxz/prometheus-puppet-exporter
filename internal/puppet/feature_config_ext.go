package puppet

import (
	"errors"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/zxzharmlesszxz/prometheus-exporter-framework/exporter/featurekit"
)

type Config struct {
	ConfigFile  string
	ConfigPath  string
	LockPath    string
	SummaryPath string

	configPathConfigured  bool
	lockPathConfigured    bool
	summaryPathConfigured bool
}

const (
	DefaultRefreshInterval = time.Minute
	DefaultConfigPath      = "/etc/puppetlabs/puppet/puppet.conf"
	DefaultLockPath        = "/opt/puppetlabs/puppet/public/agent_disabled.lock"
	DefaultSummaryPath     = "/opt/puppetlabs/puppet/public/last_run_summary.yaml"
)

var DefaultFeatureConfigFileName = "prometheus-puppet-exporter.yml"

type featureConfigFile struct {
	ConfigPath  string `yaml:"config_path"`
	LockPath    string `yaml:"lock_path"`
	SummaryPath string `yaml:"summary_path"`
}

var featureConfigFlagSpecs = []featurekit.FeatureConfigFlagSpec[Config]{
	{
		Name:    "config-path",
		Help:    "Path under which to find puppet.conf file",
		Default: DefaultConfigPath,
		Bind: func(flag *kingpin.FlagClause, config *Config) {
			flag.StringVar(&config.ConfigPath)
			flag.IsSetByUser(&config.configPathConfigured)
		},
	},
	{
		Name:    "lock-path",
		Help:    "Path under which to find agent lock file",
		Default: DefaultLockPath,
		Bind: func(flag *kingpin.FlagClause, config *Config) {
			flag.StringVar(&config.LockPath)
			flag.IsSetByUser(&config.lockPathConfigured)
		},
	},
	{
		Name:    "summary-path",
		Help:    "Path under which to find summary file",
		Default: DefaultSummaryPath,
		Bind: func(flag *kingpin.FlagClause, config *Config) {
			flag.StringVar(&config.SummaryPath)
			flag.IsSetByUser(&config.summaryPathConfigured)
		},
	},
}

func NewDefaultConfig() Config {
	return Config{
		ConfigPath:  DefaultConfigPath,
		LockPath:    DefaultLockPath,
		SummaryPath: DefaultSummaryPath,
	}
}

func FeatureConfigFile(config *Config) *string {
	return &config.ConfigFile
}

func ValidateFeatureConfig(config Config) error {
	if config.ConfigPath == "" {
		return errors.New("config path must not be empty")
	}
	if config.LockPath == "" {
		return errors.New("lock path must not be empty")
	}
	if config.SummaryPath == "" {
		return errors.New("summary path must not be empty")
	}
	return nil
}

func FeatureRuntimeConfigEntries(_ featurekit.RuntimeConfigContext[Config], config Config) []any {
	return []any{
		"config_path", normalizeString(config.ConfigPath, DefaultConfigPath),
		"lock_path", normalizeString(config.LockPath, DefaultLockPath),
		"summary_path", normalizeString(config.SummaryPath, DefaultSummaryPath),
	}
}

func normalizeString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func ResolveFeatureConfig(featureName string, config Config) (Config, string, bool, error) {
	var fileConfig featureConfigFile
	cfgFile, loaded, err := featurekit.LoadFeatureConfigFile(featureName, config.ConfigFile, &fileConfig)
	if err != nil {
		return config, cfgFile, false, err
	}
	if !loaded {
		return config, cfgFile, false, nil
	}

	if fileConfig.ConfigPath != "" && !config.configPathConfigured {
		config.ConfigPath = fileConfig.ConfigPath
	}
	if fileConfig.LockPath != "" && !config.lockPathConfigured {
		config.LockPath = fileConfig.LockPath
	}
	if fileConfig.SummaryPath != "" && !config.summaryPathConfigured {
		config.SummaryPath = fileConfig.SummaryPath
	}
	return config, cfgFile, loaded, nil
}
