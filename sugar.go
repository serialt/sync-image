package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func service() {
	slog.Debug("debug msg")
	slog.Info("info msg")
	slog.Error("error msg")

	GenerateDynamicConf()

}

// isExcludeTag 排除tag
func isExcludeTag(tag string) bool {
	match, _ := regexp.MatchString(`^[A-Za-z]+$`, tag)
	if match {
		return true
	} else if len(tag) >= 40 {
		return true
	} else {
		lowerTag := strings.ToLower(tag)

		// 如果tag包含被排除的字段，则直接返回true
		var _exclude bool
		for _, ex := range config.Exclude {
			_exclude = false
			if strings.Contains(lowerTag, ex) {
				_exclude = true
				break
			}

		}
		if _exclude {
			return true
		}
		return false
	}

}

type CTag struct {
	Tags []string `json:"tags"`
}

// GetRepoTagFromGcr 获取 gcr.io repo 最新的 tag
func GetRepoTagFromGcr(image string, limit int, host string) (tags []string, err error) {
	tag_url := fmt.Sprintf("https://%s/v2/%s/tags/list", host, image)

	tagList := new(CTag)
	resp, err := http.Get(tag_url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	json.Unmarshal(body, &tagList)
	for _, v := range tagList.Tags {
		// 如果tag不报含sig结尾
		if !strings.HasSuffix(v, "sig") {
			if image == "build-image/kube-cross" {
				tags = append(tags, v)
			} else if !isExcludeTag(v) {
				tags = append(tags, v)
			}
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(tags)))
	// 当tag数多于limit值时，不回tag进行切分
	if len(tags) > limit {
		tags = tags[:limit]
	}
	slog.Info("Get tag from gcr", "host", host,
		"image", image,
		"tags", tags,
		"err", err)

	imageName := ""
	if strings.Contains(image, "/") {
		_grcImage := strings.Split(image, "/")
		imageName = _grcImage[1]
	} else {
		imageName = image
	}

	syncedTag, _ := GetDockerTags(os.Getenv("DEST_HUB_USERNAME")+"/"+imageName, limit)
	var _tags []string
	for _, v := range tags {
		if !slices.Contains(syncedTag, v) {
			_tags = append(_tags, v)
		}
	}
	tags = _tags

	slog.Info("Get sync tag from gcr", "host", host,
		"image", image,
		"tags", tags,
		"err", err)
	return
}

type ESToken struct {
	Token string `json:"token"`
}

// GetRepoTagFromElastic 获取 elastic.io repo 最新的 tag
func GetRepoTagFromElastic(image string, limit int) (tags []string, err error) {
	tokenApiURL := fmt.Sprintf("https://docker-auth.elastic.co/auth?service=token-service&scope=repository:%s:pull", image)
	tagApiURL := fmt.Sprintf("https://docker.elastic.co/v2/%s/tags/list", image)

	resp, err := http.Get(tokenApiURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	tokenData, _ := io.ReadAll(resp.Body)
	var _esToken ESToken
	json.Unmarshal(tokenData, &_esToken)

	this_token := fmt.Sprintf("Bearer %s", _esToken.Token)

	req, _ := http.NewRequest("GET", tagApiURL, nil)
	req.Header.Add("User-Agent", "docker/19.03.12 go/go1.13.10 git-commit/48a66213fe kernel/5.8.0-1.el7.elrepo.x86_64 os/linux arch/amd64 UpstreamClient(Docker-Client/19.03.12 (linux))")
	req.Header.Add("Authorization", this_token)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var esTagList CTag
	json.Unmarshal(body, &esTagList)
	for _, v := range esTagList.Tags {
		// 如果tag不报含sig结尾
		if !strings.HasSuffix(v, "sig") {
			if !isExcludeTag(v) {
				tags = append(tags, v)
			}
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(tags)))
	// 当tag数多于limit值时，不回tag进行切分
	if len(tags) > limit {
		tags = tags[:limit]
	}
	slog.Info("Get tag from es",
		"image", image,
		"tags", tags,
		"err", err)

	imageName := ""
	if strings.Contains(image, "/") {
		_esImage := strings.Split(image, "/")
		imageName = _esImage[1]
	} else {
		imageName = image
	}

	syncedTag, _ := GetDockerTags(os.Getenv("DEST_HUB_USERNAME")+"/"+imageName, limit)
	var _tags []string
	for _, v := range tags {
		if !slices.Contains(syncedTag, v) {
			_tags = append(_tags, v)
		}
	}
	tags = _tags

	slog.Info("Get sync tag from gcr",
		"image", image,
		"tags", tags,
		"err", err)
	return
}

type SubQuayTag struct {
	Name string `json:"name"`
}

type QuayTag struct {
	Tags []SubQuayTag `json:"tags"`
}

// GetRepoTagFromQuay 获取 quay.io repo 最新的 tag
func GetRepoTagFromQuay(image string, limit int) (tags []string, err error) {
	tag_url := fmt.Sprintf("https://quay.io/api/v1/repository/%s/tag/?onlyActiveTags=true&limit=100", image)

	tagList := new(QuayTag)
	resp, err := http.Get(tag_url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &tagList)
	for _, v := range tagList.Tags {
		if !isExcludeTag(v.Name) {
			tags = append(tags, v.Name)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(tags)))
	if len(tags) > limit {
		tags = tags[:limit]
	}
	slog.Info("Get tag from quay.io",
		"image", image,
		"tags", tags,
		"err", err)

	imageName := ""
	if strings.Contains(image, "/") {
		_quayImage := strings.Split(image, "/")
		imageName = _quayImage[1]
	} else {
		imageName = image
	}

	syncedTag, _ := GetDockerTags(os.Getenv("DEST_HUB_USERNAME")+"/"+imageName, limit)
	var _tags []string
	for _, v := range tags {
		if !slices.Contains(syncedTag, v) {
			_tags = append(_tags, v)
		}
	}
	tags = _tags
	slog.Info("Get sync tag from quay.io",
		"image", image,
		"tags", tags,
		"err", err)
	return
}

type dockerToken struct {
	Token string `json:"token"`
}

func GetDockerToken() (err error) {
	username := os.Getenv("DEST_HUB_USERNAME")
	password := os.Getenv("DEST_HUB_PASSWORD")

	apiUrl := "https://hub.docker.com/v2/users/login"
	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)
	resp, err := http.PostForm(apiUrl, data)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_data, _ := io.ReadAll(resp.Body)

	var tmpT dockerToken
	json.Unmarshal(_data, &tmpT)
	tmpDockerToken = tmpT.Token
	return

}

type Results struct {
	Name string `json:"name"`
}
type DockerResp struct {
	Results []Results `json:"results"`
}

func GetDockerTags(image string, limit int) (tags []string, err error) {
	_image := strings.Split(image, "/")
	this_token := fmt.Sprintf("Bearer %s", tmpDockerToken)
	// dockerhub 支持不使用token请求，但是容易被限速:
	// curl https://hub.docker.com/v2/namespaces/serialt/repositories/vscode/tags
	apiURL := fmt.Sprintf("https://hub.docker.com/v2/namespaces/%s/repositories/%s/tags", _image[0], _image[1])
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	req.Header.Add("Accept", "application/json; charset=utf-8")
	req.Header.Add("Authorization", this_token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_data, _ := io.ReadAll(resp.Body)

	var _resp DockerResp
	json.Unmarshal(_data, &_resp)

	for _, v := range _resp.Results {
		if !isExcludeTag(v.Name) {
			tags = append(tags, v.Name)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(tags)))
	if len(tags) > limit {
		tags = tags[:limit]
	}
	slog.Info("Get tag from docker.io",
		"image", image,
		"tags", tags,
		"err", err)

	return
}

type GhcrMetaData struct {
	Container CTag `json:"container"`
}

type GhcrPkgResp struct {
	Name     string
	Metadata GhcrMetaData `json:"metadata"`
}

func GetRepoTagFromGHCR(image string, limit int) (tags []string, err error) {
	githubToken := os.Getenv("MY_GITHUB_TOKEN")
	_image := strings.Split(image, "/")
	apiURL := fmt.Sprintf("https://api.github.com/users/%s/packages/container/%s/versions?pagepage=1&per_page=1000", _image[0], _image[1])

	this_token := fmt.Sprintf("Bearer %s", githubToken)

	// 获取package tag
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", this_token)
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_ghcrPkgData, _ := io.ReadAll(resp.Body)
	var pkgNameList []GhcrPkgResp
	json.Unmarshal(_ghcrPkgData, &pkgNameList)

	for _, v := range pkgNameList {
		if len(v.Metadata.Container.Tags) != 0 {
			for _, tag := range v.Metadata.Container.Tags {
				if !isExcludeTag(tag) {
					tags = append(tags, tag)
				}
			}
		}

	}

	sort.Sort(sort.Reverse(sort.StringSlice(tags)))
	if len(tags) > limit {
		tags = tags[:limit]
	}
	slog.Info("Get tag from ghcr.io",
		"image", image,
		"tags", tags,
		"err", err)

	_ghcrImage := strings.Split(image, "/")
	imageName := _ghcrImage[1]

	syncedTag, _ := GetDockerTags(os.Getenv("DEST_HUB_USERNAME")+"/"+imageName, limit)
	var _tags []string
	for _, v := range tags {
		if !slices.Contains(syncedTag, v) {
			_tags = append(_tags, v)
		}
	}
	tags = _tags
	slog.Info("Get sync tag from ghcr.io",
		"image", image,
		"tags", tags,
		"err", err)

	return
}

type McrTag struct {
	Name string `json:"name"`
}

// GetRepoTagFromMcr 获取 mcr.microsoft.com repo 最新的 tag
func GetRepoTagFromMcr(image string, limit int) (tags []string, err error) {
	tag_url := fmt.Sprintf("https://mcr.microsoft.com/api/v1/catalog/%s/tags", image)

	var tagList []McrTag
	resp, err := http.Get(tag_url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	json.Unmarshal(body, &tagList)
	for _, v := range tagList {
		// if !isExcludeTag(v) {
		tags = append(tags, v.Name)
		// }

	}

	sort.Sort(sort.Reverse(sort.StringSlice(tags)))
	// 当tag数多于limit值时，不回tag进行切分
	if len(tags) > limit {
		tags = tags[:limit]
	}
	slog.Info("Get tag from mcr",
		"image", image,
		"tags", tags,
		"err", err)

	imageName := ""
	if strings.Contains(image, "/") {
		_mrcImage := strings.Split(image, "/")
		imageName = _mrcImage[1]
	} else {
		imageName = image
	}

	syncedTag, _ := GetDockerTags(os.Getenv("DEST_HUB_USERNAME")+"/"+imageName, limit)
	var _tags []string
	for _, v := range tags {
		if !slices.Contains(syncedTag, v) {
			_tags = append(_tags, v)
		}
	}
	tags = _tags

	slog.Info("Get sync tag from mcr",
		"image", image,
		"tags", tags,
		"err", err)
	return
}

// GetRepoTags 获取repo最新的tag
func GetRepoTags(repo, image string, limit int) (tags []string) {
	switch repo {
	case "gcr.io", "k8s.gcr.io", "registry.k8s.io":
		tags, _ = GetRepoTagFromGcr(image, limit, repo)
	case "quay.io":
		tags, _ = GetRepoTagFromQuay(image, limit)
	case "docker.io":
		tags, _ = GetDockerTags(image, limit)
	case "ghcr.io":
		tags, _ = GetRepoTagFromGHCR(image, limit)
	case "docker.elastic.co":
		tags, _ = GetRepoTagFromElastic(image, limit)
	case "mcr.microsoft.com":
		tags, _ = GetRepoTagFromMcr(image, config.McrLast)
	}

	return
}

// 生成动态同步配置
func GenerateDynamicConf() {

	SkopeoData := make(map[string]map[string]map[string][]string)
	for domain, v := range config.Images {
		imagesM := make(map[string][]string)
		imagesMap := map[string]map[string][]string{
			"images": imagesM,
		}
		for _, i := range v {

			tag := GetRepoTags(domain, i, config.Last)
			if len(tag) == 0 {
				continue
			}

			imagesM[i] = tag
			imagesMap["images"] = imagesM
			SkopeoData[domain] = imagesMap

		}

	}

	data, err := yaml.Marshal(SkopeoData)
	if err != nil {
		slog.Error("yaml marshal failed", "err", err)
		return
	}
	err = os.WriteFile(config.AutoSyncfile, data, 0644)
	if err != nil {
		slog.Error("Write auto sync data to file failed", "err", err)
	}

}
