FROM golang:alpine AS builder
WORKDIR /src
ADD . .
RUN apk add --no-cache make
RUN make build

FROM alpine:latest
COPY --from=builder /src/bin/hoarder /usr/bin/hoarder
RUN apk add --no-cache tzdata
ENTRYPOINT ["hoarder"]
