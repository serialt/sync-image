package main

var (
	// 版本信息
	appVersion bool // 控制是否显示版本
	APPVersion = "v0.0.2"
	BuildTime  = "2006-01-02 15:04:05"
	GoVersion  = "go1.21"
	GitCommit  = "xxxxxxxxxxx"
	ConfigFile = "config.yaml"
	config     *Config

	tmpDockerToken string
)

type Config struct {
	Exclude      []string            `yaml:"exclude"`
	Last         int                 `yaml:"last"`
	McrLast      int                 `yaml:"mcrLast"`
	Images       map[string][]string `yaml:"images"`
	AutoSyncfile string              `yaml:"autoSyncfile"`
}
