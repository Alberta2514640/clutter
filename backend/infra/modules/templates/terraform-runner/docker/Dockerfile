FROM hashicorp/terraform:1.7

# Install aws-cli, bash, jq via Alpine APK
RUN apk add --no-cache \
    aws-cli \
    bash \
    jq \
    curl

# Create non-root user for security
RUN addgroup -g 1000 terraform && \
    adduser -u 1000 -G terraform -h /home/terraform -D terraform

# Create working directory
WORKDIR /terraform

# Copy entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh && \
    chown -R terraform:terraform /terraform && \
    chown terraform:terraform /usr/local/bin/entrypoint.sh

# Switch to non-root user
USER terraform

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
