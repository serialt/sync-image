package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/hashicorp/go-version"
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

	logoutHub := crab.SliceMerge(config.Hub, config.DockerHub)
	for _, v := range logoutHub {
		iCMD := fmt.Sprintf("skopeo logout %s ", v.URL)
		result, err := RunCMD(iCMD)
		if err != nil {
			slog.Error("logout failed", "cmd", iCMD, "err", err)
			fmt.Println(result)
			continue
		}
		fmt.Println(result)
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
	if match || len(tag) >= 40 {
		return true
	} else {
		lowerTag := strings.ToLower(tag)
		// 如果tag包含被排除的字段，则直接返回true
		for _, ex := range config.Exclude {
			if strings.Contains(lowerTag, ex) {
				return true
			}
		}
		return false
	}

}

type CTag struct {
	Tags []string `json:"tags"`
}

// GetOCITags 获取 oci repo 最新的 tag
func GetOCITags(url, image string, limit int) (tags []string, err error) {
	allTags := GetTags(url, image)
	for _, v := range allTags {
		// 如果tag不报含sig结尾
		if !strings.HasSuffix(v, "sig") {
			if image == "build-image/kube-cross" {
				tags = append(tags, v)
			} else if !isExcludeTag(v) {
				tags = append(tags, v)
			}
		}
	}
	tags = slice.Difference(ParseVersion(tags, limit), GetExitTags(image))
	slog.Info("Get sync tag from oci", "host", url, "image", image, "tags", tags, "err", err)
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
			tag, _ := GetOCITags(domain, i, config.Last)
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
	if url != "docker.io" {
		lCMD := fmt.Sprintf("skopeo login -u %s -p %s %s", sHub.Username, sHub.Password, "docker.io")
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

	destHub := url + "/" + username
	iCMD = fmt.Sprintf("skopeo --insecure-policy sync -a  --src yaml --dest docker %s %s", skopeoFile, destHub)

	result, err = RunCMD(iCMD)
	if err != nil {
		slog.Error("sync image failed", "file", skopeoFile, "destHub", destHub, "err", err)
		fmt.Println(result)
		return
	}
	slog.Info("skopeo sync succeed", "url", url, "user", username, "file", skopeoFile)
	fmt.Println(result)

	return
}

type RepositoryTag struct {
	Repository string   `json:"repository"`
	Tags       []string `json:"tags"`
}

func GetTags(url, image string) (tags []string) {

	listCMD := fmt.Sprintf("skopeo list-tags  docker://%v/%v", url, image)
	slog.Info("run cmd", "cmd", listCMD)
	result, err := RunCMD(listCMD)
	if err != nil {
		slog.Error("get tags by skopeo failed", "err", err)
		return
	}
	var rTag RepositoryTag
	err = json.Unmarshal([]byte(result), &rTag)
	if err != nil {
		slog.Error("json unmarshal failed", "err", err)
		return
	}
	tags = rTag.Tags
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

	return GetTags(config.Hub[0].URL, config.Hub[0].Username+"/"+eImage)
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

func RunCommandWithTimeout(timeout int, command string, args ...string) (stdout, stderr string, isKilled bool) {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command(command, args...)

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Start()
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()
	after := time.After(time.Duration(timeout) * time.Second)
	select {
	case <-after:
		cmd.Process.Signal(syscall.SIGINT)
		time.Sleep(10 * time.Millisecond)
		cmd.Process.Kill()
		isKilled = true
	case <-done:
		isKilled = false
	}
	stdout = string(bytes.TrimSpace(stdoutBuf.Bytes())) // Remove \n
	stderr = string(bytes.TrimSpace(stderrBuf.Bytes())) // Remove \n
	return
}

func ParseVersion(versionsRaw []string, count int) (tags []string) {
	slog.Info("Get tags", "tags", versionsRaw)
	vLen := len(versionsRaw)
	if vLen == 0 {
		return
	}
	versions := make([]*version.Version, vLen)
	for i, raw := range versionsRaw {
		v, _ := version.NewVersion(raw)
		versions[i] = v
	}
	sort.Sort(version.Collection(versions))
	if vLen > count {
		versions = versions[vLen-count:]
	}
	for _, v := range versions {
		tags = append(tags, v.String())
	}
	return
}
