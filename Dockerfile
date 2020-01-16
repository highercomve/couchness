FROM golang:alpine as src

ENV GO111MODULES=on

WORKDIR /app/
COPY . .

RUN apk update; apk add git
RUN go get

FROM src as builder 

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64

RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} go build -o /go/bin/couchness -v .

FROM alpine

COPY --from=builder /go/bin /pkg/bin

RUN chmod 755 /pkg/bin/couchness

ENV USER root

ENTRYPOINT [ "/pkg/bin/couchness" ]
