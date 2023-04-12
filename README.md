# sync_image

[![sync](https://github.com/serialt/sync_image/actions/workflows/sync.yml/badge.svg?branch=main)](https://github.com/serialt/sync_image/actions/workflows/sync.yml)

Synchronize container image

## 使用

仓库使用 `Github Action` 每天自动运行脚本同步镜像到 `Docker Hub` 和 阿里云。

动态同步的镜像列表。
> 默认获取最新的 5 个 tag 用于同步。

```
k8s.gcr.io/etcd
k8s.gcr.io/kube-proxy
k8s.gcr.io/kube-apiserver
k8s.gcr.io/kube-scheduler
k8s.gcr.io/kube-controller-manager
k8s.gcr.io/coredns/coredns
k8s.gcr.io/dns/k8s-dns-node-cache
k8s.gcr.io/ingress-nginx/controller
k8s.gcr.io/metrics-server/metrics-server
k8s.gcr.io/ingress-nginx/kube-webhook-certgen
k8s.gcr.io/kube-state-metrics/kube-state-metrics
k8s.gcr.io/sig-storage/nfs-subdir-external-provisioner
k8s.gcr.io/sig-storage/csi-node-driver-registrar
k8s.gcr.io/sig-storage/csi-provisioner
k8s.gcr.io/sig-storage/csi-resizer
k8s.gcr.io/sig-storage/csi-snapshotter
k8s.gcr.io/sig-storage/csi-attacher
```


静态同步的镜像列表。
> 使用指定的 tag 用于同步。

```
k8s.gcr.io/pause
k8s.gcr.io/defaultbackend-amd64
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



## 文件介绍

- `config.yaml`: 供 `generate_sync_yaml.py` 脚本使用，此文件配置了需要动态(获取`last`个最新的版本)同步的镜像列表。
- `custom_sync.yaml`: 自定义的 [`skopeo`](https://github.com/containers/skopeo) 同步源配置文件。
- `generate_sync_yaml.py`: 根据配置，动态生成 [`skopeo`](https://github.com/containers/skopeo) 同步源配置文件。
- `sync.sh`: 用于执行同步操作。
