FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
WORKDIR /app
ARG TARGETOS
ARG TARGETARCH
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /beancs-controller ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /beancs-controller /beancs-controller
EXPOSE 8080
CMD ["/beancs-controller"]
