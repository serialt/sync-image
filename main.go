package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/serialt/sugar/v3"
)

func init() {
	flag.BoolVar(&appVersion, "v", false, "Version messages")
	flag.StringVar(&ConfigFile, "c", "config.yaml", "Config file")
	flag.Parse()

	err := sugar.LoadConfig(ConfigFile, &config)
	if err != nil {
		config = new(Config)
	}
	slog.SetDefault(sugar.New())
	if len(config.Hub) == 0 {
		hubUsernames := strings.Split(os.Getenv("HUB_USERNAME"), ",")
		hubPasswords := strings.Split(os.Getenv("HUB_PASSWORD"), ",")
		hubURLs := strings.Split(os.Getenv("HUB_URL"), ",")
		for i, v := range hubUsernames {
			config.Hub = append(config.Hub, DockerHub{
				URL:      hubURLs[i],
				Username: v,
				Password: hubPasswords[i],
			})
		}
	}
	config.GithubToken = os.Getenv("MY_GITHUB_TOKEN")
	if len(config.DockerHub) == 0 {
		hubUsernames := strings.Split(os.Getenv("DOCKER_HUB_USERNAME"), ",")
		hubPasswords := strings.Split(os.Getenv("DOCKER_HUB_PASSWORD"), ",")
		for i, v := range hubUsernames {
			config.DockerHub = append(config.Hub, DockerHub{
				URL:      "docker.io",
				Username: v,
				Password: hubPasswords[i],
			})
		}
	}

}
func main() {
	if appVersion {
		fmt.Printf("AppVersion: %v\nGo Version: %v\nBuild Time: %v\nGit Commit: %v\n\n",
			APPVersion,
			GoVersion,
			BuildTime,
			GitCommit,
		)
		return
	}

	service()

}
