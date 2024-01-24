#
# Copyright (c) 2023 Intel Corporation
# Copyright (c) 2021 Starblaze Ltd
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

ARG BASE=golang:1.21-alpine3.18
FROM --platform=$BUILDPLATFORM ${BASE} AS builder
ARG TARGETARCH

ARG ADD_BUILD_TAGS=""
ARG MAKE="make -e ADD_BUILD_TAGS=$ADD_BUILD_TAGS build"
ARG ALPINE_PKG_BASE="make git openssh-client"
ARG ALPINE_PKG_EXTRA=""

RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add --update --no-cache ${ALPINE_PKG_BASE} ${ALPINE_PKG_EXTRA}

WORKDIR /device-lora-go

COPY go.mod vendor* ./

# 引入本地私有仓库中的工具代码库
# RUN git config --global url."http://git:git@192.168.0.10/iot-sdk".insteadOf "http://192.168.0.10/iot-sdk"
# RUN go env -w GOARCH=$TARGETARCH GOPROXY=https://goproxy.cn,direct GOPRIVATE=192.168.0.10 GOINSECURE=192.168.0.10
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN [ ! -d "vendor" ] && go mod download all || echo "skipping..."

COPY . .
RUN $MAKE

FROM alpine:3.18

LABEL license='SPDX-License-Identifier: Apache-2.0' \
  copyright='Copyright (c) 2023: Intel'

LABEL Name=device-lora-go Version=${VERSION}

RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
# dumb-init needed for injected secure bootstrapping entrypoint script when run in secure mode.
RUN apk add --update --no-cache dumb-init

COPY --from=builder /device-lora-go/cmd /
COPY --from=builder /device-lora-go/LICENSE /
COPY --from=builder /device-lora-go/Attribution.txt /

EXPOSE 59986

ENTRYPOINT ["/device-lora"]
CMD ["--cp=consul://edgex-core-consul:8500", "--registry"]