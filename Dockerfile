FROM node:20-alpine AS ui
WORKDIR /src/core/ui
RUN corepack enable
COPY core/ui/package.json core/ui/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY core/ui/ ./
RUN pnpm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY adaptor/go/go.mod adaptor/go/go.sum* ./adaptor/go/
WORKDIR /src/adaptor/go
RUN go mod download
WORKDIR /src
COPY adaptor/go/ ./adaptor/go/
COPY core/migrations/ ./adaptor/go/migrations/
COPY --from=ui /src/core/ui/dist ./adaptor/go/ui/dist
WORKDIR /src/adaptor/go
RUN go build -o /out/microscope ./cmd/server

FROM alpine:3
COPY --from=build /out/microscope /usr/local/bin/microscope
EXPOSE 8093
ENTRYPOINT ["microscope"]
