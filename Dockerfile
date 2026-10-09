# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom

# Go and Node versions are supplied by Mise through the release workflow or local container task.
ARG GO_VERSION
ARG NODE_VERSION
ARG DBIP_RELEASE=2026-08

# ---- Go base --------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS go-base

# ---- GeoIP databases -----------------------------------------------------
FROM go-base AS geoip
ARG DBIP_RELEASE
COPY tools/geoipdb/main.go /geoipdb.go
RUN go run /geoipdb.go -release "${DBIP_RELEASE}" -output /geoip

# ---- Web build ------------------------------------------------------------
# Build the frontend bundle so the Go stage can embed it. The runtime image
# does not include Node.
FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS web
WORKDIR /workspace/web

# Install dependencies against the lockfile first for layer caching.
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN npm install --global "$(node --print 'require("./package.json").packageManager')"
RUN pnpm install --frozen-lockfile

COPY web/ ./
COPY schema/ ../schema/
RUN pnpm build

# ---- Go build -------------------------------------------------------------
FROM go-base AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN apk add --no-cache upx
WORKDIR /workspace

# Cache module downloads before copying source.
COPY go.mod go.sum ./
RUN go mod download
ARG GO_LICENSES_VERSION
RUN go install github.com/google/go-licenses/v2@${GO_LICENSES_VERSION}

COPY cmd/ cmd/
COPY internal/ internal/
COPY web/ web/

# Overlay the freshly built frontend bundle so go:embed uses the real assets.
COPY --from=web /workspace/web/dist web/dist

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go-licenses save ./cmd/woodstar --save_path third_party_licenses --ignore github.com/woodleighschool/woodstar --force

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X github.com/woodleighschool/woodstar/internal/buildinfo.Version=${VERSION#v}" -o woodstar ./cmd/woodstar
RUN upx --best --lzma woodstar
RUN mkdir /data

# ---- Runtime --------------------------------------------------------------
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY LICENSE /LICENSE
COPY --from=builder /workspace/third_party_licenses /third_party_licenses
COPY --from=builder /usr/local/go/LICENSE /third_party_licenses/go/LICENSE
COPY --from=builder /workspace/woodstar /woodstar
COPY --from=geoip /geoip/dbip-city-lite.mmdb /share/geoip/dbip-city-lite.mmdb
COPY --from=geoip /geoip/dbip-asn-lite.mmdb /share/geoip/dbip-asn-lite.mmdb
COPY --from=builder --chown=65532:65532 /data /data
ENV WOODSTAR_GEOIP_CITY_FILE=/share/geoip/dbip-city-lite.mmdb \
    WOODSTAR_GEOIP_ASN_FILE=/share/geoip/dbip-asn-lite.mmdb
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/woodstar"]
