FROM golang:1.22-bullseye AS build

RUN apt update; apt install -y libvips-dev

ARG GIT_SHA=none
ARG GIT_BRANCH=none
ARG RELEASE=none

WORKDIR /app
COPY . .
RUN go build -ldflags="-X 'main.GitHash=$GIT_SHA' -X 'main.GitBranch=$GIT_BRANCH' -X 'main.BuildTime=`date`'" -o /backend

FROM linuxserver/ffmpeg:7.0.1

RUN apt update; apt install -y libvips42

WORKDIR /
COPY --from=build /backend /backend
ENTRYPOINT ["/backend"]