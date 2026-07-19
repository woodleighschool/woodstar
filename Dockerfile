ARG NODE_VERSION=26.5.0
ARG GO_VERSION=1.26.5
FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS web

WORKDIR /workspace/web
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN npm install --global "$(node -p 'require("./package.json").packageManager')"
RUN pnpm install --frozen-lockfile
COPY web/ ./
COPY schema/ ../schema/
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=0.0.0-dev

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY web/ web/
COPY --from=web /workspace/web/dist web/dist

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X github.com/woodleighschool/woodstar/internal/buildinfo.Version=${VERSION}" -o woodstar ./cmd/woodstar

FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/woodstar /woodstar
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/woodstar"]
CMD ["serve"]
