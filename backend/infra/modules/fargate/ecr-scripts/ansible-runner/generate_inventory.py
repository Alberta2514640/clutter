#!/usr/bin/env python3
"""
Generate an Ansible inventory YAML file from EC2 instance IDs.

Queries the EC2 API for instance details and writes a YAML inventory
that uses the aws_ssm connection plugin (SSM Session Manager) instead of SSH.

Usage:
    python3 generate_inventory.py \
        --instance-ids "i-0123456789abcdef0,i-0987654321fedcba0" \
        --region us-east-1 \
        --output /playbooks/inventory.yml \
        --aws-access-key-id AKIA... \
        --aws-secret-access-key ... \
        --aws-session-token ... \
        --s3-bucket my-ssm-bucket
"""

import argparse
import sys
import yaml
import boto3
from botocore.exceptions import ClientError


def get_instance_details(instance_ids: list[str], region: str, session: boto3.Session) -> dict[str, dict]:
    """
    Query EC2 for instance details (ID, private IP, state).

    Returns:
        dict mapping instance_id -> { 'private_ip': str, 'state': str }
    """
    ec2 = session.client('ec2', region_name=region)

    instances = {}
    try:
        paginator = ec2.get_paginator('describe_instances')
        for page in paginator.paginate(InstanceIds=instance_ids):
            for reservation in page.get('Reservations', []):
                for instance in reservation.get('Instances', []):
                    instance_id = instance['InstanceId']
                    state = instance.get('State', {}).get('Name', 'unknown')

                    if state != 'running':
                        print(
                            f"[generate_inventory] WARNING: Instance {instance_id} is not running "
                            f"(state: {state})",
                            file=sys.stderr,
                        )
                        continue

                    instances[instance_id] = {
                        'private_ip': instance.get('PrivateIpAddress', ''),
                        'state': state,
                    }
    except ClientError as e:
        print(f"[generate_inventory] ERROR: Failed to describe instances: {e}", file=sys.stderr)
        sys.exit(1)

    return instances


def generate_inventory(
    instances: dict[str, dict],
    region: str,
    s3_bucket: str,
    access_key_id: str,
    secret_access_key: str,
    session_token: str,
    remote_user: str = 'ec2-user',
) -> dict:
    """
    Build an Ansible inventory structure using the aws_ssm connection plugin.

    SSM Session Manager does not require SSH keys or open inbound ports.
    The target EC2 instances only need the SSM Agent running and an
    instance profile with AmazonSSMManagedInstanceCore policy.

    Client credentials are injected as ansible_aws_ssm_* vars. The SSM
    connection plugin uses these credentials for BOTH SSM StartSession AND
    S3 file staging. The client role must therefore have S3 permissions on
    the Clutter bucket (GetBucketLocation, PutObject, GetObject, DeleteObject).
    """
    hosts = {}
    for instance_id, details in instances.items():
        host_vars = {
            'ansible_host': instance_id,  # SSM uses instance ID, not IP
            'ansible_connection': 'community.aws.aws_ssm',
            'ansible_aws_ssm_region': region,
            'ansible_user': remote_user,
            # S3 bucket the SSM plugin uses for module file staging.
            # S3 ops from the plugin run via the per-host client credentials below.
            'ansible_aws_ssm_bucket_name': s3_bucket,
            # Per-host SSM credentials (client assumed role) — used for both
            # SSM StartSession AND S3 file staging by the connection plugin.
            # The client role must have s3:GetBucketLocation, s3:PutObject,
            # s3:GetObject, s3:DeleteObject on the Clutter S3 bucket.
            'ansible_aws_ssm_access_key_id': access_key_id,
            'ansible_aws_ssm_secret_access_key': secret_access_key,
            'ansible_aws_ssm_session_token': session_token,
            # Bucket policy enforces SSE on all PutObject requests.
            # The SSM plugin uploads module files using the client role, so
            # it must include the SSE header or the policy will deny the upload.
            'ansible_aws_ssm_bucket_sse_mode': 'AES256',
        }
        hosts[instance_id] = host_vars

    group_vars = {
        'ansible_connection': 'community.aws.aws_ssm',
        'ansible_aws_ssm_region': region,
        'ansible_user': remote_user,
    }
    inventory = {
        'all': {
            'hosts': hosts,
            'vars': group_vars,
        },
    }

    return inventory


def main():
    parser = argparse.ArgumentParser(description='Generate Ansible inventory from EC2 instance IDs')
    parser.add_argument(
        '--instance-ids',
        required=True,
        help='Comma-separated list of EC2 instance IDs',
    )
    parser.add_argument(
        '--region',
        required=True,
        help='AWS region',
    )
    parser.add_argument(
        '--output',
        required=True,
        help='Output file path for the inventory YAML',
    )
    parser.add_argument(
        '--remote-user',
        default='ec2-user',
        help='SSM user for Ansible (default: ec2-user)',
    )
    parser.add_argument(
        '--aws-access-key-id',
        required=True,
        help='AWS access key ID for the client role',
    )
    parser.add_argument(
        '--aws-secret-access-key',
        required=True,
        help='AWS secret access key for the client role',
    )
    parser.add_argument(
        '--aws-session-token',
        required=True,
        help='AWS session token for the client role',
    )
    parser.add_argument(
        '--s3-bucket',
        required=True,
        help='S3 bucket name for Ansible SSM plugin file staging',
    )
    args = parser.parse_args()

    # Parse instance IDs
    instance_ids = [iid.strip() for iid in args.instance_ids.split(',') if iid.strip()]
    if not instance_ids:
        print('[generate_inventory] ERROR: No valid instance IDs provided', file=sys.stderr)
        sys.exit(1)

    print(f'[generate_inventory] Looking up {len(instance_ids)} instance(s) in {args.region}...')

    # Create a boto3 session with the client's assumed-role credentials for EC2 describe
    client_session = boto3.Session(
        aws_access_key_id=args.aws_access_key_id,
        aws_secret_access_key=args.aws_secret_access_key,
        aws_session_token=args.aws_session_token,
    )

    # Query EC2 using client credentials (their account)
    instances = get_instance_details(instance_ids, args.region, client_session)

    if not instances:
        print('[generate_inventory] ERROR: No reachable instances found', file=sys.stderr)
        sys.exit(1)

    print(f'[generate_inventory] Found {len(instances)} running instance(s)')

    # Generate and write inventory
    inventory = generate_inventory(
        instances,
        args.region,
        args.s3_bucket,
        args.aws_access_key_id,
        args.aws_secret_access_key,
        args.aws_session_token,
        args.remote_user,
    )

    try:
        with open(args.output, 'w') as f:
            yaml.dump(inventory, f, default_flow_style=False)
        print(f'[generate_inventory] Inventory written to {args.output}')
    except OSError as e:
        print(f'[generate_inventory] ERROR: Failed to write inventory to {args.output}: {e}', file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
