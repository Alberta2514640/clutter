FROM python:3.11-slim AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget \
    git \
    unzip \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir \
    ansible-core==2.17.7 \
    awscli==1.31.0 \
    boto3==1.34.0 \
    botocore==1.34.0 \
    psycopg2-binary==2.9.9 \
    pyyaml==6.0.1 \
    jinja2==3.1.2

# Install the Ansible AWS collection (includes aws_ssm connection plugin)
RUN ansible-galaxy collection install amazon.aws:==8.2.1 community.aws:==8.0.0

# -------------------------------------------------------------------
FROM python:3.11-slim

# Install only runtime dependencies (including postgresql-client for psql)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    postgresql-client \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install SSM Session Manager plugin in runtime stage
# Using pinned version with GPG signature verification for reproducibility and security
RUN SSM_PLUGIN_VERSION="2.4.680.0" && \
    curl -o /tmp/session-manager-plugin.deb \
    "https://s3.amazonaws.com/session-manager-downloads/plugin/${SSM_PLUGIN_VERSION}/ubuntu_64bit/session-manager-plugin.deb" \
    && curl -o /tmp/session-manager-plugin.deb.sig \
    "https://s3.amazonaws.com/session-manager-downloads/plugin/${SSM_PLUGIN_VERSION}/ubuntu_64bit/session-manager-plugin.deb.sig" \
    && curl -o /tmp/session-manager-plugin.pub \
    "https://s3.amazonaws.com/session-manager-downloads/plugin/public-key/session-manager-plugin.pub" \
    && apt-get install -y --no-install-recommends gnupg \
    && gpg --import /tmp/session-manager-plugin.pub \
    && gpg --verify /tmp/session-manager-plugin.deb.sig /tmp/session-manager-plugin.deb \
    && dpkg -i /tmp/session-manager-plugin.deb \
    && rm /tmp/session-manager-plugin.deb /tmp/session-manager-plugin.deb.sig /tmp/session-manager-plugin.pub \
    && apt-get remove -y gnupg && apt-get autoremove -y && rm -rf /var/lib/apt/lists/*

# Copy installed Python packages from builder
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Copy Ansible collections from builder
COPY --from=builder /root/.ansible /root/.ansible

# Create working directories
RUN mkdir -p /playbooks /tmp/ansible /var/log/ansible

# Copy scripts
COPY entrypoint.sh /entrypoint.sh
COPY generate_inventory.py /usr/local/bin/generate_inventory.py

RUN chmod +x /entrypoint.sh /usr/local/bin/generate_inventory.py

WORKDIR /playbooks
ENTRYPOINT ["/entrypoint.sh"]
