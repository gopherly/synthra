# Build the webapp example from the synthra module root:
#   docker build -f examples/Dockerfile -t synthra-webapp-example .
#
# The image runs with defaults from examples/webapp/config.yaml unless
# you pass -e WEBAPP_* overrides.

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/webapp ./examples/webapp

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/webapp ./webapp
COPY examples/webapp/config.yaml ./config.yaml
ENTRYPOINT ["./webapp"]
