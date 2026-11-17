FROM hashicorp/terraform:1.14.4

# Install required tools
RUN apk add --no-cache \
    bash \
    jq \
    aws-cli

WORKDIR /app

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
