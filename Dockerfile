# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

FROM golang:1.23 AS builder
ARG VERSION

ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux

WORKDIR /src
COPY . .

RUN useradd -u 10007 -s /bin/false tfciuser
RUN mkdir -p /etc/ssl/certs && update-ca-certificates

RUN go build \
  -ldflags "-X 'github.com/hashicorp/tfci/version.Version=$VERSION' -s -w -extldflags '-static'" \
  -o /bin/app \
  .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

COPY --from=builder /bin/app /usr/local/bin/tfci

USER tfciuser

ENV TF_LOG=INFO \
  TF_FORMAT=CONSOLE

ENTRYPOINT ["/usr/local/bin/tfci"]
