FROM --platform=$BUILDPLATFORM golang:alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
ADD . .
RUN apk add --no-cache make curl
RUN curl -o selenium.jar https://selenium-release.storage.googleapis.com/3.14/selenium-server-standalone-3.14.0.jar
RUN go install github.com/ogen-go/ogen/cmd/ogen@v0.76.0     # this is stupid
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make bin

FROM instrumentisto/geckodriver:latest
COPY --from=builder /src/bin/* /usr/local/bin/
COPY --from=builder /src/docker/init /
COPY --from=builder /src/selenium.jar /
RUN apt-get update
RUN apt-get install -y --no-install-recommends --no-install-suggests tzdata openjdk-21-jre-headless
ENTRYPOINT ["/init"]
