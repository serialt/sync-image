#!/bin/bash
serialt

hub="registry.cn-hangzhou.aliyuncs.com"
repo="$hub/serialt"
dockerhub="serialt"


if [ -f sync.yaml ]; then
   echo "[Start] sync......."
   
   sudo skopeo login -u ${HUB_USERNAME} -p ${HUB_PASSWORD} ${hub} \
   && sudo skopeo --insecure-policy sync --src yaml --dest docker sync.yaml ${repo} \
   && sudo skopeo --insecure-policy sync --src yaml --dest docker custom_sync.yaml ${repo}

   sudo skopeo login -u ${DOCKER_HUB_USERNAME} -p ${HUB_PASSWORD} \
   && sudo skopeo --insecure-policy sync --src yaml --dest docker sync.yaml ${dockerhub} \
   && sudo skopeo --insecure-policy sync --src yaml --dest docker custom_sync.yaml ${dockerhub}


   echo "[End] done."
   
else
    echo "[Error]not found sync.yaml!"
fi
