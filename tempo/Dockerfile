# STAGE 1: The Builder
FROM busybox:1.36-uclibc AS builder

RUN mkdir -p /data/traces && \
    chown -R 10001:10001 /data

# STAGE 2: The Final Image
FROM grafana/tempo:latest

# Copy pre-chowned directory
COPY --from=builder --chown=10001:10001 /data /data

USER 10001
VOLUME ["/data"]