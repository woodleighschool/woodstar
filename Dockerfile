FROM node:24-alpine AS web

WORKDIR /workspace/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack prepare --activate
RUN pnpm install --frozen-lockfile
COPY web/ ./
COPY schema/ ../schema/
RUN pnpm build

FROM golang:1.26 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY web/ web/
COPY --from=web /workspace/web/dist web/dist

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w" -o woodstar ./cmd/woodstar

FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/woodstar /woodstar
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/woodstar"]
CMD ["serve"]
