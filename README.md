# sync-image
[![sync](https://github.com/serialt/sync-image/actions/workflows/sync.yml/badge.svg?branch=main)](https://github.com/serialt/sync-image/actions/workflows/sync.yml)


> Synchronize container image

### 项目介绍
* 基于Go重写`sync-image`,感谢[lework](https://github.com/lework/sync_image)。
* 支持GCR、MCR、elastic、quay.io、docker.io、registry.k8s.io、ghcr.io 镜像同步到 docker hub 和阿里云。

环境变量
```shell
DEST_HUB_USERNAME
DEST_HUB_PASSWORD
MY_GITHUB_TOKEN
```

配置文件
config.yaml

```yaml
# 普通镜像同步个数
last: 10  
# mcr镜像同步个数
mcrLast: 50
# 自定义的skopeo同步镜像配置文件
customfile: custom_sync.yaml
# 动态生成的skopeo同步镜像配置文件
autoSyncfile: sync.yaml
# tag中含有以下关键字不同步
exclude:
  - 'alpha'
  - 'beta' 
  - 'rc' 
  - 'amd64'
  - 'ppc64le' 
  - 'arm64' 
  - 'arm' 
  - 's390x'   
  - 'SNAPSHOT' 
  - 'snapshot'
  - 'debug' 
  - 'master' 
  - 'latest' 
  - 'main'
  - 'sig'
  - 'sha'
  - 'mips'
images:
  docker.elastic.co:
    - elasticsearch/elasticsearch
    - app-search/app-search
  quay.io:
    - coreos/flannel
    - ceph/ceph
    - cephcsi/cephcsi
  k8s.gcr.io:
    - metrics-server/metrics-server
    - kube-state-metrics/kube-state-metrics
  registry.k8s.io:
    - metrics-server/metrics-server
    - pause
    - etcd
    - coredns/coredns
    - build-image/kube-cross
  gcr.io:
    - kaniko-project/executor
  ghcr.io:
    - k3d-io/k3d-tools
    - k3d-io/k3d-proxy
    - kube-vip/kube-vip
  mcr.microsoft.com:
    - devcontainers/base
    - devcontainers/go
  docker.io:
    - flannel/flannel
    - flannel/flannel-cni-plugin
    - calico/kube-controllers

```


静态同步的镜像列表。
> 使用指定的 tag 用于同步。
`custom_sync.yaml`
```yaml
ghcr.io:
  images:
    kube-vip/kube-vip:
    - 'v0.6.0'
    - 'v0.4.4'
    k3d-io/k3d-tools:
    - '5.5.2'
```

同步规则

```bash
# docker hub
k8s.gcr.io/{image_name}  ==>  docker.io/serialt/{image_name}

# aliyun
k8s.gcr.io/{image_name}  ==>  registry.cn-hangzhou.aliyuncs.com/serialt/{image_name}
```

**拉取镜像**

```bash
#  docker hub
$ docker pull serialt/kube-scheduler:[image_tag]

# aliyun
$ docker pull registry.cn-hangzhou.aliyuncs.com/serialt/kube-scheduler:[image_tag]
```

**搜索镜像**

[Docker Hub](https://hub.docker.com/u/serialt)



### 文件介绍

- `config.yaml`: 供 `sync-image` 使用，此文件配置了需要动态(获取`last`个最新的版本)同步的镜像列表。
- `custom_sync.yaml`: 自定义的 [`skopeo`](https://github.com/containers/skopeo) 同步源配置文件。
- `sync.sh`: 用于执行同步操作。