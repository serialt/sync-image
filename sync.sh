#!/bin/bash

hub="docker.io"
repo="$hub/${DEST_HUB_USERNAME}"

hub2="registry.cn-hangzhou.aliyuncs.com"
repo2="$hub2/${DEST_HUB_USERNAME}"



if [ -f sync.yaml ]; then
   echo "[Start] sync......."
   
    sudo skopeo login -u ${DEST_HUB_USERNAME} -p ${DEST_HUB_PASSWORD} ${hub} \
    && sudo skopeo --insecure-policy sync -a --src yaml --dest docker sync.yaml ${repo} \
    && sudo skopeo --insecure-policy sync -a --src yaml --dest docker custom_sync.yaml ${repo}
    sleep 3
    sudo skopeo login -u ${DEST_HUB_USERNAME} -p ${DEST_HUB_PASSWORD} ${hub2} \
    && sudo skopeo --insecure-policy sync -a --src yaml --dest docker sync.yaml ${repo2} \
    && sudo skopeo --insecure-policy sync -a  --src yaml --dest docker custom_sync.yaml ${repo2}


   echo "[End] done."
   
else
    echo "[Error]not found sync.yaml!"
fi
