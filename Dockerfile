FROM golang:alpine AS builder
WORKDIR /src
ADD . .
RUN apk add --no-cache make
RUN make bin

FROM alpine:latest
COPY --from=builder /src/bin/* /usr/local/bin/
RUN apk add --no-cache tzdata
ENTRYPOINT ["hoarder"]
