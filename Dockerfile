FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/feedreader ./cmd/feedreader

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
ENV FEEDREADER_DB_PATH=/data/feedreader.db \
    FEEDREADER_HOST=0.0.0.0 \
    FEEDREADER_PORT=8080
COPY --from=builder --chown=65532:65532 /out/feedreader /feedreader
COPY --from=builder --chown=65532:65532 /src/web /app/web
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/feedreader"]
CMD ["serve", "--host", "0.0.0.0", "--port", "8080"]
