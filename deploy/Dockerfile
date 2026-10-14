FROM golang:1.25.6-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ARG TARGET=./gateway/cmd/server
RUN CGO_ENABLED=0 go build -trimpath -o /service ${TARGET}

FROM alpine:3.22
RUN apk add --no-cache ca-certificates ffmpeg && adduser -D -u 10001 service
WORKDIR /app
COPY --from=build /service /app/service
USER service
ENTRYPOINT ["/app/service"]
