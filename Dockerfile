FROM --platform=$BUILDPLATFORM golang:alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
ADD . .
RUN apk add --no-cache make
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make bin

FROM alpine:latest
COPY --from=builder /src/bin/* /usr/local/bin/
RUN apk add --no-cache tzdata
ENTRYPOINT ["hoarder"]
