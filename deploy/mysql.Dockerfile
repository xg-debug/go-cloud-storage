ARG MYSQL_IMAGE=mysql:8.0
FROM ${MYSQL_IMAGE}
# Copy into the image to avoid executable-bit/noexec differences on host mounts.
COPY --chmod=0644 deploy/init-services.sh /docker-entrypoint-initdb.d/01-services.sh
