ARG BUILDER_IMAGE=docker.io/library/golang
ARG BUILDER_VERSION=1.20-alpine

FROM $BUILDER_IMAGE:$BUILDER_VERSION AS builder

WORKDIR /go/src/app

ARG GOPRIVATE="github.com/atomyze-foundation/*"
ARG NETRC="machine github.com login USERNAME password TOKEN"
ARG VERSION=unknown

RUN echo "$NETRC" > ~/.netrc

COPY go.mod go.sum ./
# hadolint ignore=DL3018
RUN apk add --no-cache git=~2 binutils=~2 upx>=3
RUN CGO_ENABLED=0 go mod download

COPY . .

RUN CGO_ENABLED=0 go build -v -ldflags="-X 'main.AppInfoVer=$VERSION'" -o "/go/bin/robot" && \
    strip "/go/bin/robot" && \
    upx -5 -q "/go/bin/robot"

FROM docker.io/library/alpine:3.18
COPY --chown=65534:65534 --from=builder "/go/bin/robot" /
USER 65534

ENTRYPOINT [ "/robot" ]
