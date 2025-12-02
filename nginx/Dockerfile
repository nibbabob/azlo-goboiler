FROM nginx:1.29-alpine

# Remove default assets
RUN rm -rf /usr/share/nginx/html/*

# Copy configuration
COPY nginx/nginx.conf /etc/nginx/nginx.conf


# Permissions
RUN mkdir -p /etc/nginx/certs && chown -R 101:101 /etc/nginx/certs

USER 101