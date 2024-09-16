package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/serialt/crab"
	"gopkg.in/yaml.v3"
)

func service() {

	GenerateDynamicConf()

	files, err := crab.FileLoopFiles(config.WorkDir)
	if err != nil {
		slog.Error("Get skopeo files failed", "err", err)
		os.Exit(5)
	}
	c := SyncClient{}
	for _, iFile := range files {
		for _, v := range config.Hub {
			c.Next()
			SkopeoSync(c.Hub, v.Username, v.Password, v.URL, iFile)
		}

	}

}

func (s *SyncClient) Next() {
	if len(config.DockerHub) == 1 {
		s.Hub = DockerHub{
			URL:      config.DockerHub[0].URL,
			Username: config.DockerHub[0].Username,
			Password: config.DockerHub[0].Password,
		}
	} else {
		index := int(s.Index) % (len(config.DockerHub) - 1)
		s.Hub = config.DockerHub[index]
		index++
		s.Index = int64(index)

	}

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

	syncedTag := GetExitTags(image)
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

	syncedTag := GetExitTags(image)
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

	syncedTag := GetExitTags(image)
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

type GhcrMetaData struct {
	Container CTag `json:"container"`
}

type GhcrPkgResp struct {
	Name     string
	Metadata GhcrMetaData `json:"metadata"`
}

func GetRepoTagFromGHCR(image string, limit int) (tags []string, err error) {
	_image := strings.Split(image, "/")
	apiURL := fmt.Sprintf("https://api.github.com/users/%s/packages/container/%s/versions?pagepage=1&per_page=1000", _image[0], _image[1])

	this_token := fmt.Sprintf("Bearer %s", config.GithubToken)

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
	slog.Info("Get tag from ghcr.io", "image", image, "tags", tags)

	syncedTag := GetExitTags(image)
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

// GetRepoTags 获取repo最新的tag
func GetRepoTags(repo, image string, limit int) (tags []string) {

	switch repo {
	case "gcr.io", "k8s.gcr.io":
		tags, _ = GetRepoTagFromGcr(image, limit, repo)
	case "quay.io":
		tags, _ = GetRepoTagFromQuay(image, limit)
	case "ghcr.io":
		tags, _ = GetRepoTagFromGHCR(image, limit)
	case "docker.elastic.co":
		tags, _ = GetRepoTagFromElastic(image, limit)

	default:
		// mcr.microsoft.com
		// registry.k8s.io
		// docker.io
		tags = GetTags(repo, image, limit)
	}

	return
}

// 生成动态同步配置
func GenerateDynamicConf() {

	// SkopeoData := make(map[string]map[string]map[string][]string)
	for domain, v := range config.Images {

		for _, i := range v {
			SkopeoData := make(map[string]map[string]map[string][]string)
			imagesM := make(map[string][]string)
			imagesMap := map[string]map[string][]string{
				"images": imagesM,
			}
			// tag := GetRepoTags(domain, i, config.Last)
			tag := GetRepoTags(domain, i, config.Last)
			if len(tag) == 0 {
				continue
			}

			imagesM[i] = tag
			imagesMap["images"] = imagesM
			SkopeoData[domain] = imagesMap

			// 生成多个同步文件
			SkopeoImageData := make(map[string]map[string]map[string][]string)
			SkopeoImageData[domain] = imagesMap
			data, err := yaml.Marshal(SkopeoImageData)
			if err != nil {
				slog.Error("yaml marshal failed", "err", err)
				return
			}
			filenameSlice := strings.Split(i, "/")
			filename := strings.Join(filenameSlice, "-")

			err = os.WriteFile(config.WorkDir+"/"+filename+".yaml", data, 0644)
			if err != nil {
				slog.Error("Write auto sync data to file failed", "err", err)
			}

		}

	}

	// data, err := yaml.Marshal(SkopeoData)
	// if err != nil {
	// 	slog.Error("yaml marshal failed", "err", err)
	// 	return
	// }
	// err = os.WriteFile(config.AutoSyncfile, data, 0644)
	// if err != nil {
	// 	slog.Error("Write auto sync data to file failed", "err", err)
	// }

}

func SkopeoSync(sHub DockerHub, username, password, url, skopeoFile string) {
	// docker.io
	// registry.cn-hangzhou.aliyuncs.com
	// swr.cn-east-3.myhuaweicloud.com
	iUrl := strings.Split(url, ".")
	iCMD := ""
	// destHub := fmt.Sprintf("%s/%s", url, username)
	switch iUrl[len(iUrl)-2] {
	case "myhuaweicloud":
		iCMD = fmt.Sprintf("skopeo login -u %s@%s -p %s %s", username, iUrl[0], password, url)
	default:
		iCMD = fmt.Sprintf("skopeo login -u %s -p %s %s", username, password, url)
	}
	if sHub.URL == "docker.io" {
		lCMD := fmt.Sprintf("skopeo login -u %s -p %s %s", username, password, url)
		result, err := RunCMD(lCMD)
		fmt.Println(result)
		if err != nil {
			slog.Error("login docker.io failed", "err", err)
			return
		}
	}

	result, err := RunCMD(iCMD)

	if err != nil {
		slog.Error("login failed", "url", url, "file", skopeoFile, "err", err)
		fmt.Println(result)
		return
	}
	slog.Info("login hub", "url", url, "user", username)
	fmt.Println(result)

	// iCMD = fmt.Sprintf("skopeo --insecure-policy sync -a  --src yaml --dest docker %s %s", skopeoFile, destHub)
	// result, err = RunCMD(iCMD)
	// if err != nil {
	// 	slog.Error("sync image failed", "file", skopeoFile, "destHub", destHub, "err", err)
	// 	fmt.Println(result)
	// 	return
	// }
	// slog.Info("skopeo sync succeed", "url", url, "user", username, "file", skopeoFile)
	// fmt.Println(result)

	iCMD = fmt.Sprintf("skopeo logout %s ", url)
	result, err = RunCMD(iCMD)
	if err != nil {
		slog.Error("logout failed", "cmd", iCMD, "err", err)
		fmt.Println(result)
		return
	}
	fmt.Println(result)
	return
}

func GetTags(url, image string, limit int) (tags []string) {
	var srcTags []string
	var hubURL string
	switch url {
	case "docker.io":
		hubURL = "https://registry-1.docker.io"
	default:
		hubURL = fmt.Sprintf("https://%s", url)
	}
	hub, err := registry.New(hubURL, "", "")
	if err != nil {
		slog.Error("create hub failed", "hub", hubURL, "err", err)
		return
	}
	hubTags, err := hub.Tags(image)
	if err != nil {
		slog.Error("get tag failed", "hub", hubURL, "image", image, "err", err)
	}

	if url != "mcr.microsoft.com" {
		for _, v := range hubTags {
			if !isExcludeTag(v) {
				srcTags = append(srcTags, v)
			}
		}
	}

	if len(srcTags) > limit {
		srcTags = srcTags[:limit]
	}
	slog.Info("get tag", "url", hubURL, "image", image, "tags", srcTags)

	etags := GetExitTags(image)
	for _, tag := range srcTags {
		if !slices.Contains(etags, tag) {
			tags = append(tags, tag)
		}
	}
	slog.Info("get tag", "url", url, "image", image, "tags", tags)
	return
}

func GetExitTags(image string) (tags []string) {
	// get exits tag
	var eImage string
	eImageS := strings.Split(image, "/")
	if len(eImageS) == 2 {
		eImage = eImageS[1]
	} else {
		eImage = image
	}
	var hubURL, username, password string
	switch config.Hub[0].URL {
	case "docker.io":
		hubURL = "https://registry-1.docker.io"
		username = config.Hub[0].Username
		password = config.Hub[0].Password
	default:
		hubURL = fmt.Sprintf("https://%s", config.Hub[0].URL)
	}

	eHub, err := registry.New(hubURL, username, password)
	if err != nil {
		slog.Error("get exits hub failed", "hub", hubURL, "err", err)
	}
	tags, err = eHub.Tags(config.Hub[0].Username + "/" + eImage)
	if err != nil {
		slog.Error("get exits tag from hub failed", "url", hubURL, "image", eImage, "err", err)

	}
	slog.Info("get exits tag from hub", "url", hubURL, "image", eImage, "tags", tags)

	return
}

func RunCMD(str string, workDir ...string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", str)
	if len(workDir) > 0 {
		cmd.Dir = workDir[0]
	}
	result, err := cmd.CombinedOutput()
	if err != nil {
		return string(result), err
	}
	return string(result), nil
}
