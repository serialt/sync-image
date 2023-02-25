import os
import re
import yaml
import requests
from json import dumps as jsondumps
from distutils.version import LooseVersion

# 基本配置
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
CONFIG_FILE = os.path.join(BASE_DIR, 'config.yaml')
SYNC_FILE = os.path.join(BASE_DIR, 'sync.yaml')
CUSTOM_SYNC_FILE = os.path.join(BASE_DIR, 'custom_sync.yaml')


def is_exclude_tag(tag):
    """
    排除tag
    :param tag:
    :return:
    """
    excludes = [
        'alpha', 'beta', 'rc', 'amd64', 'ppc64le', 'arm64', 'arm', 's390x',
        'SNAPSHOT', 'debug', 'master', 'latest', 'main'
    ]

    for e in excludes:
        if e.lower() in tag.lower():
            return True
        if str.isalpha(tag):
            return True
        if len(tag) >= 40:
            return True

    # 处理带有 - 字符的 tag
    if re.search("-\d$", tag, re.M | re.I):
        return False
    if '-' in tag:
        return True

    return False


def get_repo_gcr_tags(image, limit=5, host="k8s.gcr.io"):
    """
    获取 gcr.io repo 最新的 tag
    :param host:
    :param image:
    :param limit:
    :return:
    """

    hearders = {
        'User-Agent':
        'docker/19.03.12 go/go1.13.10 git-commit/48a66213fe kernel/5.8.0-1.el7.elrepo.x86_64 os/linux arch/amd64 UpstreamClient(Docker-Client/19.03.12 \(linux\))'
    }

    tag_url = "https://{host}/v2/{image}/tags/list".format(host=host,
                                                           image=image)

    tags = []
    tags_data = []
    manifest_data = []

    try:
        tag_rep = requests.get(url=tag_url, headers=hearders)
        tag_req_json = tag_rep.json()
        manifest_data = tag_req_json['manifest']
    except Exception as e:
        print('[Get tag Error]', e)
        return tags

    for manifest in manifest_data:
        sha256_data = manifest_data[manifest]
        sha256_tag = sha256_data.get('tag', [])
        if len(sha256_tag) > 0:
            # 排除 tag
            if is_exclude_tag(sha256_tag[0]):
                continue
            tags_data.append({
                'tag':
                sha256_tag[0],
                'timeUploadedMs':
                sha256_data.get('timeUploadedMs')
            })
    tags_sort_data = sorted(tags_data,
                            key=lambda i: i['timeUploadedMs'],
                            reverse=True)

    # limit tag
    tags_limit_data = tags_sort_data[:limit]

    image_docker_tags = get_docker_io_tags(os.environ['DEST_HUB_USERNAME'],
                                           image, 0)
    for t in tags_limit_data:
        # 去除同步过的
        if t['tag'] in image_docker_tags:
            continue

        tags.append(t['tag'])

    print('[repo tag]', tags)
    return tags


def get_repo_quay_tags(image, limit=5):
    """
    获取 quay.io repo 最新的 tag
    :param image:
    :param limit:
    :return:
    """

    hearders = {
        'User-Agent':
        'docker/19.03.12 go/go1.13.10 git-commit/48a66213fe kernel/5.8.0-1.el7.elrepo.x86_64 os/linux arch/amd64 UpstreamClient(Docker-Client/19.03.12 \(linux\))'
    }

    tag_url = "https://quay.io/api/v1/repository/{image}/tag/?onlyActiveTags=true&limit=100".format(
        image=image)

    tags = []
    tags_data = []
    manifest_data = []

    try:
        tag_rep = requests.get(url=tag_url, headers=hearders)
        tag_req_json = tag_rep.json()
        manifest_data = tag_req_json['tags']
    except Exception as e:
        print('[Get tag Error]', e)
        return tags

    for manifest in manifest_data:
        name = manifest.get('name', '')

        # 排除 tag
        if is_exclude_tag(name):
            continue

        tags_data.append({'tag': name, 'start_ts': manifest.get('start_ts')})

    tags_sort_data = sorted(tags_data,
                            key=lambda i: i['start_ts'],
                            reverse=True)

    # limit tag
    tags_limit_data = tags_sort_data[:limit]

    image_docker_tags = get_docker_io_tags(os.environ['DEST_HUB_USERNAME'],
                                           image, 0)
    for t in tags_limit_data:
        # 去除同步过的
        if t['tag'] in image_docker_tags:
            continue

        tags.append(t['tag'])

    print('[repo tag]', tags)
    return tags


def get_docker_token(username, password):
    baseUrl = "https://hub.docker.com/v2/users/login"
    header = {}
    header['Accept'] = 'application/json'
    header['Content-Type'] = 'application/json'
    data = jsondumps({"username": username, "password": password})
    response = requests.post(baseUrl, data=data, headers=header)
    dockerhub_token = ""
    try:
        response = response.json()
        dockerhub_token = response['token']
    except requests.exceptions.RequestException as e:
        raise SystemExit(e)

    return dockerhub_token


def get_docker_io_tags(namespace, image, limit=5):
    username = os.environ['DEST_HUB_USERNAME']
    password = os.environ['DEST_HUB_PASSWORD']
    token = get_docker_token(username, password)
    this_token = "Bearer {docker_token}".format(docker_token=token)
    hearders = {
        "Content-Type": "application/json; charset=utf-8",
        "Accept": "application/json; charset=utf-8",
        "Authorization": this_token
    }
    image_name = image.split('/')[-1]
    tag_url = "https://hub.docker.com/v2/namespaces/{username}/repositories/{image}/tags".format(
        username=namespace, image=image_name)
    print(tag_url)

    tags = []
    tags_data = []
    manifest_data = []

    try:
        tag_rep = requests.get(url=tag_url, headers=hearders)
        tag_req_json = tag_rep.json()
        manifest_data = tag_req_json['results']
    except Exception as e:
        print('[Get tag Error]', e)
        tags = ['latest']
        return tags
    for tag in manifest_data:
        name = tag.get('name', '')

        # 排除 tag
        if is_exclude_tag(name):
            continue

        tags_data.append(name)

    tags_sort_data = sorted(tags_data, key=LooseVersion, reverse=True)

    if limit != 0:
        # limit tag
        tags_limit_data = tags_sort_data[:limit]

        image_docker_tags = get_docker_io_tags(os.environ['DEST_HUB_USERNAME'],
                                               image, 0)
        for t in tags_limit_data:
            # 去除同步过的
            if t in image_docker_tags:
                continue

            tags.append(t)
    else:
        tags = tags_sort_data

    print('[repo tag]', tags)
    return tags


def get_repo_elastic_tags(image, limit=5):
    """
    获取 elastic.io repo 最新的 tag
    :param image:
    :param limit:
    :return:
    """
    token_url = "https://docker-auth.elastic.co/auth?service=token-service&scope=repository:{image}:pull".format(
        image=image)
    tag_url = "https://docker.elastic.co/v2/{image}/tags/list".format(
        image=image)

    tags = []
    tags_data = []
    manifest_data = []

    hearders = {
        'User-Agent':
        'docker/19.03.12 go/go1.13.10 git-commit/48a66213fe kernel/5.8.0-1.el7.elrepo.x86_64 os/linux arch/amd64 UpstreamClient(Docker-Client/19.03.12 \(linux\))'
    }

    try:
        token_res = requests.get(url=token_url, headers=hearders)
        token_data = token_res.json()
        access_token = token_data['token']
    except Exception as e:
        print('[Get repo token]', e)
        return tags

    hearders['Authorization'] = 'Bearer ' + access_token

    try:
        tag_rep = requests.get(url=tag_url, headers=hearders)
        tag_req_json = tag_rep.json()
        manifest_data = tag_req_json['tags']
    except Exception as e:
        print('[Get tag Error]', e)
        return tags

    for tag in manifest_data:
        # 排除 tag
        if is_exclude_tag(tag):
            continue
        tags_data.append(tag)

    tags_sort_data = sorted(tags_data, key=LooseVersion, reverse=True)

    # limit tag
    tags_limit_data = tags_sort_data[:limit]

    image_docker_tags = get_docker_io_tags(os.environ['DEST_HUB_USERNAME'],
                                           image, 0)
    for t in tags_limit_data:
        # 去除同步过的
        if t in image_docker_tags:
            continue

        tags.append(t)

    print('[repo tag]', tags)
    return tags


def get_repo_tags(repo, image, limit=5):
    """
    获取 repo 最新的 tag
    :param repo:
    :param image:
    :param limit:
    :return:
    """
    tags_data = []
    if repo == 'gcr.io':
        tags_data = get_repo_gcr_tags(image, limit, "gcr.io")
    elif repo == 'k8s.gcr.io':
        tags_data = get_repo_gcr_tags(image, limit, "k8s.gcr.io")
    elif repo == 'registry.k8s.io':
        tags_data = get_repo_gcr_tags(image, limit, "registry.k8s.io")
    elif repo == 'quay.io':
        tags_data = get_repo_quay_tags(image, limit)
    elif repo == 'docker.io':
        tags_data = get_docker_io_tags(os.environ['SRC_HUB_USERNAME'], image,
                                       limit)
    elif repo == 'docker.elastic.co':
        tags_data = get_repo_elastic_tags(image, limit)
    return tags_data


def generate_dynamic_conf():
    """
    生成动态同步配置
    :return:
    """

    print('[generate_dynamic_conf] start.')
    config = None
    with open(CONFIG_FILE, 'r') as stream:
        try:
            config = yaml.safe_load(stream)
        except yaml.YAMLError as e:
            print('[Get Config]', e)
            exit(1)

    print('[config]', config)

    skopeo_sync_data = {}

    for repo in config['images']:
        if repo not in skopeo_sync_data:
            skopeo_sync_data[repo] = {'images': {}}
        if config['images'][repo] is None:
            continue
        for image in config['images'][repo]:
            print("[image] {image}".format(image=image))
            sync_tags = get_repo_tags(repo, image, config['last'])
            if len(sync_tags) > 0:
                skopeo_sync_data[repo]['images'][image] = sync_tags
            # skopeo_sync_data[repo]['images'][image].append('latest')
            else:
                print('[{image}] no sync tag.'.format(image=image))

    print('[sync data]', skopeo_sync_data)

    with open(SYNC_FILE, 'w+') as f:
        yaml.safe_dump(skopeo_sync_data, f, default_flow_style=False)

    print('[generate_dynamic_conf] done.', end='\n\n')


def generate_custom_conf():
    """
    生成自定义的同步配置
    :return:
    """

    print('[generate_custom_conf] start.')
    custom_sync_config = None
    with open(CUSTOM_SYNC_FILE, 'r') as stream:
        try:
            custom_sync_config = yaml.safe_load(stream)
        except yaml.YAMLError as e:
            print('[Get Config]', e)
            exit(1)

    print('[custom_sync config]', custom_sync_config)

    custom_skopeo_sync_data = {}

    for repo in custom_sync_config:
        if repo not in custom_skopeo_sync_data:
            custom_skopeo_sync_data[repo] = {'images': {}}
        if custom_sync_config[repo]['images'] is None:
            continue
        for image in custom_sync_config[repo]['images']:
            image_docker_tags = get_docker_io_tags(image)
            for tag in custom_sync_config[repo]['images'][image]:
                if tag in image_docker_tags:
                    continue
                if image not in custom_skopeo_sync_data[repo]['images']:
                    custom_skopeo_sync_data[repo]['images'][image] = [tag]
                else:
                    custom_skopeo_sync_data[repo]['images'][image].append(tag)

    print('[custom_sync data]', custom_skopeo_sync_data)

    with open(CUSTOM_SYNC_FILE, 'w+') as f:
        yaml.safe_dump(custom_skopeo_sync_data, f, default_flow_style=False)

    print('[generate_custom_conf] done.', end='\n\n')


generate_dynamic_conf()
generate_custom_conf()
