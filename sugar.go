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

func isMatch(tag string) bool {
	if len(tag) > 15 {
		return false
	}
	match, _ := regexp.MatchString(`^(v\d+|\d+)[a-zA-Z0-9.-]*$`, tag)
	return match
}
func isExcludeTag(tag string) bool {
	match, _ := regexp.MatchString(config.Regexp, tag)
	return match
}

// // isExcludeTag 排除tag
// func isExcludeTag(tag string, isExclude []string) bool {

// 	lowerTag := strings.ToLower(tag)
// 	// 如果tag包含被排除的字段，则直接返回true
// 	for _, ex := range isExclude {
// 		if strings.Contains(lowerTag, ex) {
// 			return true
// 		}
// 	}

// 	return false

// }

// GetOCITags 获取 oci repo 最新的 tag
func GetOCITags(url, image string) (tags []string, err error) {
	allTags := GetTags(url, image)
	for _, v := range allTags {
		if isMatch(v) && (!isExcludeTag(v)) {
			tags = append(tags, v)
			// if !isExcludeTag(v, config.Exclude) {
			// 	tags = append(tags, v)
			// }
		}

	}

	var eImage string
	eImageS := strings.Split(image, "/")
	if len(eImageS) == 2 {
		eImage = eImageS[1]
	} else {
		eImage = image
	}
	var eData []string
	fData, err := os.ReadFile(config.SyncedDir + "/" + eImage + ".json")
	if err == nil {
		var rTag RepositoryTag
		err = json.Unmarshal(fData, &rTag)
		if err != nil {
			slog.Error("json unmarshal failed", "err", err)
		}
		eData = rTag.Tags
	}
	tags = crab.SliceDiff(tags, eData)
	tags = crab.SliceDiff(tags, GetExitTags(image))
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
			var tag []string
			if config.GenSynced {
				GenSyncedImages(domain, i, config.SyncedDir)
			} else {
				tag, _ = GetOCITags(domain, i)
			}
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

func ParseVersion(versions []string, count int) (tags []string) {
	vLen := len(versions)
	if vLen == 0 {
		return
	}
	versionsGo := make([]*version.Version, len(versions))
	for i, raw := range versions {
		v, _ := version.NewVersion(raw)
		versionsGo[i] = v
	}
	sort.Sort(version.Collection(versionsGo))
	// if vLen > count {
	// 	versions = versions[vLen-count:]
	// }

	return
}

func GenSyncedImages(url, image string, dir string) {

	// get exits tag
	var eImage string
	eImageS := strings.Split(image, "/")
	if len(eImageS) == 2 {
		eImage = eImageS[1]
	} else {
		eImage = image
	}
	listCMD := fmt.Sprintf("skopeo list-tags  docker://%v/%v ", url, image)
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
	rTag.Repository = fmt.Sprintf("%v/%v", url, image)
	var newTags []string
	for _, v := range rTag.Tags {
		if isMatch(v) && (!isExcludeTag(v)) {
			newTags = append(newTags, v)
		}

	}
	rTag.Tags = newTags
	jsonData, _ := json.Marshal(rTag)
	slog.Info("get synced data", "data", string(jsonData))
	tagFile := fmt.Sprintf("%v/%v.json", dir, eImage)
	os.WriteFile(tagFile, jsonData, 0644)
	return
}
