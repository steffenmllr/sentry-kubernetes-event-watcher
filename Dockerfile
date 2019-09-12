# Stage 1: Build executable
FROM golang:1.12 as buildImage

RUN mkdir -p /sentry-kubernetes-event-watcher
WORKDIR /sentry-kubernetes-event-watcher
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o watcher

# Stage 2: Create release image
FROM alpine:3
RUN apk --no-cache add ca-certificates

RUN mkdir /sentry-k8s
WORKDIR /sentry-k8s

COPY --from=buildImage /sentry-kubernetes-event-watcher/watcher /sentry-k8s/watcher
RUN chmod +x /sentry-k8s/watcher

CMD ["/sentry-k8s/watcher"]
