# Official MinIO source, pinned independently of unavailable registry images.
# Source and license: https://github.com/minio/minio/tree/7aac2a2c5b7c882e68c1ce017d8256be2feea27f
FROM golang:1.25.6-alpine AS build
ADD --checksum=sha256:71794c2df26aad0cc99e8421c58b7aa2dd55969f979b0e7d1e931042e9fabcd6 https://codeload.github.com/minio/minio/tar.gz/7aac2a2c5b7c882e68c1ce017d8256be2feea27f /tmp/source.tar.gz
RUN mkdir /src && tar -xzf /tmp/source.tar.gz --strip-components=1 -C /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -trimpath -o /minio .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates curl && adduser -D -u 10001 minio && mkdir /data && chown minio /data
COPY --from=build /minio /usr/local/bin/minio
COPY --from=build /src/LICENSE /usr/share/licenses/minio/LICENSE
USER minio
EXPOSE 9000 9001
ENTRYPOINT ["minio"]
CMD ["server", "/data", "--console-address", ":9001"]
