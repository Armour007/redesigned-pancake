## Multi-stage build for AURA backend
FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/aura ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/aura /app/aura
ENV PORT=8080
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app/aura"]
