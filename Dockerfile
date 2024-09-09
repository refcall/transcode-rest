FROM golang:1.22-bullseye AS build

RUN apt update; apt install -y libvips-dev

ARG GIT_SHA=none
ARG GIT_BRANCH=none
ARG RELEASE=none

WORKDIR /app
COPY go.mod .
COPY go.sum .
COPY *.go .
RUN go build -ldflags="-X 'main.GitHash=$GIT_SHA' -X 'main.GitBranch=$GIT_BRANCH' -X 'main.BuildTime=`date`'" -o /transcode-rest

FROM linuxserver/ffmpeg:7.0.1

RUN apt update; apt install -y libvips42

WORKDIR /
COPY --from=build /transcode-rest /transcode-rest
ENTRYPOINT ["/transcode-rest"]