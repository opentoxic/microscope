FROM node:20-alpine AS ui
WORKDIR /src/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
COPY --from=ui /src/ui/dist ./ui/dist
RUN go build -o /out/microscope ./cmd/server

FROM alpine:3
COPY --from=build /out/microscope /usr/local/bin/microscope
EXPOSE 8093
ENTRYPOINT ["microscope"]
