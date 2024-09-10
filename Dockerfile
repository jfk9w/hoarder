FROM --platform=$BUILDPLATFORM golang:alpine3.19 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
ADD . .
RUN apk add --no-cache make
RUN go install github.com/ogen-go/ogen/cmd/ogen@v0.76.0     # this is stupid
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make bin

FROM alpine:latest
COPY --from=builder /src/bin/* /usr/local/bin/
COPY --from=builder /src/docker/init /
RUN apk add --no-cache tzdata chromium chromium-chromedriver
ENTRYPOINT ["/init"]
