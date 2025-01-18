FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /main .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /main /main
ENV TRACKED_CONTAINER=
ENV RESTARTED_CONTAINER=
ENV SLEEP_INTERVAL=10
ENV CPU_THRESHOLD=80.0
CMD ["/main"]