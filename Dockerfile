FROM --platform=$BUILDPLATFORM docker.io/library/golang:alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
ADD . .
RUN apk add --no-cache make curl
RUN curl -o selenium.jar https://selenium-release.storage.googleapis.com/3.14/selenium-server-standalone-3.14.0.jar
RUN go install github.com/ogen-go/ogen/cmd/ogen@v0.76.0     # this is stupid
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make bin

FROM docker.io/instrumentisto/geckodriver:latest
COPY --from=builder /src/bin/* /usr/local/bin/
COPY --from=builder /src/docker/init /
COPY --from=builder /src/selenium.jar /
COPY docker/russian-trusted-root-ca.crt /usr/local/share/ca-certificates/russian-trusted-root-ca.crt
RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \
        ca-certificates \
        libnss3-tools \
        openjdk-21-jre-headless \
        tzdata \
    && update-ca-certificates \
    && mkdir -p /opt/hoarder/firefox-profile \
    && certutil -N --empty-password -d sql:/opt/hoarder/firefox-profile \
    && certutil -A \
        -d sql:/opt/hoarder/firefox-profile \
        -n "Russian Trusted Root CA" \
        -t "C,," \
        -i /usr/local/share/ca-certificates/russian-trusted-root-ca.crt \
    && certutil -V \
        -d sql:/opt/hoarder/firefox-profile \
        -n "Russian Trusted Root CA" \
        -u L \
    && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["/init"]
