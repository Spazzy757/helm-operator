# Build the manager binary
FROM golang:1.12.5 as builder

WORKDIR /workspace
ARG HELM_VERSION=v2.14.1
RUN echo "HELM_VERSION: ${HELM_VERSION}" \
  && curl -LO https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz \
  && tar -zxvf helm-${HELM_VERSION}-linux-amd64.tar.gz \
  && mv linux-amd64/* . \
  && chmod +x helm \
  && chmod +x tiller \
  && ./helm init --client-only \
  && echo $HOME
RUN mkdir charts
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM  python:2.7-slim

RUN apt-get update && apt-get install -y \
  ca-certificates \
  git \
  jq \
  openssh-client \
  dnsutils \
  libnss-wrapper \
  curl \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/helm .
COPY --from=builder /root/.helm .helm
COPY --from=builder /workspace/charts charts
ENV HOME=/
ENV PATH=/:$PATH
ENTRYPOINT ["/manager"]
