FROM nginx:1.29-alpine

# Remove default assets
RUN rm -rf /usr/share/nginx/html/*

# Use 'nginx/' prefix because build context is the repo root
COPY nginx/nginx.conf /etc/nginx/nginx.conf

# Copy website
COPY www /var/www/html

# Permissions
RUN mkdir -p /etc/nginx/certs && chown -R 101:101 /etc/nginx/certs

USER 101