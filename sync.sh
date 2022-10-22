#!/bin/bash

hub="docker.io"
repo="$hub/serialt"

hub2="registry.cn-hangzhou.aliyuncs.com"
repo2="$hub2/serialt"



if [ -f sync.yaml ]; then
   echo "[Start] sync......."
   
    sudo skopeo login -u ${HUB_USERNAME} -p ${HUB_PASSWORD} ${hub} \
    && sudo skopeo --insecure-policy sync -a --src yaml --dest docker sync.yaml ${repo} \
    && sudo skopeo --insecure-policy sync -a --src yaml --dest docker custom_sync.yaml ${repo}
    sleep 3
    sudo skopeo login -u ${HUB_USERNAME} -p ${HUB_PASSWORD} ${hub2} \
    && sudo skopeo --insecure-policy sync -a --src yaml --dest docker sync.yaml ${repo2} \
    && sudo skopeo --insecure-policy sync -a  --src yaml --dest docker custom_sync.yaml ${repo2}


   echo "[End] done."
   
else
    echo "[Error]not found sync.yaml!"
fi
