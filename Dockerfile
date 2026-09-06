FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server && mkdir -p /data && chown 65532:65532 /data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /server /server
COPY --from=builder --chown=65532:65532 /data /data
VOLUME ["/data"]
EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/server"]
