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

type SyncClient struct {
	Hub   DockerHub
	Index int64
}
type RepositoryInfo struct {
	Repository string   `json:"Repository"`
	Tags       []string `json:"Tags"`
}

type DockerHub struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	Exclude      []string            `yaml:"exclude"`
	Last         int                 `yaml:"last"`
	McrLast      int                 `yaml:"mcrLast"`
	Images       map[string][]string `yaml:"images"`
	AutoSyncfile string              `yaml:"autoSyncfile"`
	DockerHub    []DockerHub         `yaml:"dockerHub"`
	Hub          []DockerHub         `yaml:"hub"`
	WorkDir      string              `yaml:"workDir"`
	GithubToken  string              `yaml:"githubToken"`
	GenSynced    bool                `yaml:"genSynced"`
}
