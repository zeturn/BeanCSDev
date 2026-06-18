FROM --platform=$BUILDPLATFORM node:26-alpine AS web-builder
WORKDIR /app
COPY internal/web/package*.json ./
RUN npm ci
COPY internal/web ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
WORKDIR /app
ARG TARGETOS
ARG TARGETARCH
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /beancs-controller ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /beancs-controller /beancs-controller
EXPOSE 8080
CMD ["/beancs-controller"]
