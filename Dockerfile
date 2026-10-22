FROM --platform=$BUILDPLATFORM golang:1.26.10-alpine AS build
ARG TARGETOS TARGETARCH TARGETVARIANT VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .
RUN set -eu; \
    goarm=""; \
    if [ "${TARGETARCH}" = "arm" ] && [ -n "${TARGETVARIANT:-}" ]; then \
      goarm="${TARGETVARIANT#v}"; \
    fi; \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" GOARM="${goarm}" \
      go build -mod=vendor \
      -ldflags="-X main.Version=${VERSION}" \
      -o /iptv-tunerr ./cmd/iptv-tunerr

FROM debian:bookworm-slim
RUN set -eu; \
    attempt=1; \
    while ! apt-get -o Acquire::Retries=3 update \
      || ! DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl ffmpeg wget; do \
      if [ "$attempt" -ge 3 ]; then \
        echo "Unable to install container runtime packages from Debian mirrors" >&2; \
        exit 1; \
      fi; \
      echo "Debian package install failed (attempt ${attempt}/3); retrying after backoff"; \
      rm -rf /var/lib/apt/lists/*; \
      sleep "$((attempt * 5))"; \
      attempt=$((attempt + 1)); \
    done; \
    rm -rf /var/lib/apt/lists/*
RUN set -eu; \
    groupadd --system --gid 10001 iptvtunerr; \
    useradd --system --uid 10001 --gid 10001 --home-dir /var/lib/iptvtunerr \
      --no-create-home --shell /usr/sbin/nologin iptvtunerr; \
    install -d -m 0750 -o 10001 -g 10001 /var/lib/iptvtunerr /var/cache/iptvtunerr
COPY --from=build /iptv-tunerr /usr/local/bin/iptv-tunerr
WORKDIR /var/lib/iptvtunerr
ENV HOME=/var/lib/iptvtunerr
VOLUME ["/var/lib/iptvtunerr"]
EXPOSE 5004
USER 10001:10001
ENTRYPOINT ["iptv-tunerr"]
# Default: run tuner (refresh catalog, health check, serve). Override: e.g. docker run ... iptv-tunerr:local serve -addr :5004
CMD ["run", "-addr", ":5004"]
