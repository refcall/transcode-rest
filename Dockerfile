FROM golang:1.22-bullseye AS build

ARG COMMIT_SHA=none
ARG COMMIT_BRANCH=none
ARG RELEASE=none

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-X 'main.GitHash=$COMMIT_SHA' -X 'main.GitBranch=$COMMIT_BRANCH' -X 'main.BuildTime=`date`'" -o /backend

FROM linuxserver/ffmpeg:7.0.1

WORKDIR /
COPY --from=build /backend /backend
USER nonroot:nonroot
ENTRYPOINT ["/backend"]