#!/usr/bin/env bash

# Copyright 2020 TiKV Project Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

export QINIU_ACCESS_KEY=${QINIU_ACCESS_KEY:-}
export QINIU_SECRET_KEY=${QINIU_SECRET_KEY:-}
export QINIU_BUCKET_NAME=${QINIU_BUCKET_NAME:-charts}
export RELEASE_TAG=${RELEASE_TAG:-}
export DRY_RUN=${DRY_RUN:-}

if [ -z "$QINIU_ACCESS_KEY" ]; then
    echo "error: QINIU_ACCESS_KEY is required"
    exit 1
fi

if [ -z "$QINIU_SECRET_KEY" ]; then
    echo "error: QINIU_SECRET_KEY is required"
    exit 1
fi

if [ -z "$RELEASE_TAG" ]; then
    echo "error: RELEASE_TAG is required"
    exit 1
fi

tmpdir=$(mktemp -d)
trap "rm -rf $tmpdir" EXIT

echo "info: temporary directory is $tmpdir"

cd $tmpdir

cp -r $ROOT/charts/ .
for chart in tikv-operator; do
    echo "info: publish chart $chart"
    sed -i "s/version:.*/version: ${RELEASE_TAG}/g" charts/$chart/Chart.yaml
    sed -i "s/appVersion:.*/appVersion: ${RELEASE_TAG}/g" charts/$chart/Chart.yaml
    chartPrefixName=$chart-${RELEASE_TAG}
    tar -zcf ${chartPrefixName}.tgz -C charts $chart
    sha256sum ${chartPrefixName}.tgz > ${chartPrefixName}.sha256
    if [ -n "$DRY_RUN" ]; then
        echo "info: DRY_RUN is set, skipping uploading charts"
    else
        $ROOT/hack/upload-qiniu.py ${chartPrefixName}.tgz ${chartPrefixName}.tgz
        $ROOT/hack/upload-qiniu.py ${chartPrefixName}.sha256 ${chartPrefixName}.sha256
    fi
done

# TODO check if it's semantic version
if [ "${RELEASE_TAG}" != "latest" ]; then
    echo "info: updating the index.yaml"
    hack::ensure_helm
    curl http://charts.pingcap.org/index.yaml -o index-old.yaml
    $HELM_BIN repo index . --url http://charts.pingcap.org/ --merge index-old.yaml
    echo "info: the diff of index.yaml"
    diff -u index-old.yaml index.yaml || true
    if [ -n "$DRY_RUN" ]; then
        echo "info: DRY_RUN is set, skipping uploading index.yaml"
    else
        $ROOT/hack/upload-qiniu.py index.yaml index.yaml
    fi
else
    echo "info: RELEASE_TAG is ${RELEASE_TAG}, skip adding it into chart index file"
fi
