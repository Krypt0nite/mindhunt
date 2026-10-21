# Build the web client
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN NEXT_TELEMETRY_DISABLED=1 npx next build

# Build the server
FROM golang:1.24-alpine AS server
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /mindhunt ./cmd/mindhunt

FROM gcr.io/distroless/static-debian12
COPY --from=server /mindhunt /mindhunt
COPY --from=web /web/out /web
COPY assets/ /assets
ENV WEB_DIR=/web ASSETS_DIR=/assets PORT=8080
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/mindhunt"]
