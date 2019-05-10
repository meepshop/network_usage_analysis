#!/bin/bash
set -e
set -o pipefail

# set -x
# VER="164.0.0"
# mkdir -p /tmp
# curl -L -o /tmp/gcloud-$VER.tgz https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-$VER-linux-x86_64.tar.gz
# tar -xz -C /tmp -f /tmp/gcloud-$VER.tgz

# cat $GCLOUD_SERVICE_KEY >> /tmp/gcloud-service-key.json
# /tmp/google-cloud-sdk/gcloud --quiet components update
# /tmp/google-cloud-sdk/gcloud components install kubectl --quiet
# /tmp/google-cloud-sdk/gcloud auth activate-service-account --key-file /tmp/gcloud-service-key.json
# /tmp/google-cloud-sdk/gcloud config set project $GCLOUD_PROJECT_ID


# install kubectl
curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.7.3/bin/linux/amd64/kubectl
chmod +x ./kubectl
mv ./kubectl /tmp/

# generate kubernetes config file
/tmp/kubectl config set-cluster default-cluster \
       --insecure-skip-tls-verify=true \
       --server=https://$KUBERNETES_PUBLIC_ADDRESS:443 \
       --kubeconfig=/tmp/kubeconfig
/tmp/kubectl config set-credentials $KUBERNETES_USER \
    --username=$KUBERNETES_USER \
    --password=$KUBERNETES_PASSWD \
    --kubeconfig=/tmp/kubeconfig
/tmp/kubectl config set-context default-cluster \
    --cluster=default-cluster \
    --user=$KUBERNETES_USER \
    --kubeconfig=/tmp/kubeconfig
/tmp/kubectl config use-context default-cluster --kubeconfig=/tmp/kubeconfig

# kubernetes deploy
/tmp/kubectl apply --force -f ./kubernetes/cm_stage.yml --kubeconfig=/tmp/kubeconfig
/tmp/kubectl apply --force -f ./kubernetes/cron.yml --kubeconfig=/tmp/kubeconfig