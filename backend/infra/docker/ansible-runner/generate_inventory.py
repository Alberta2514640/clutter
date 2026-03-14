#!/usr/bin/env python3
"""
Generate an Ansible inventory YAML file from EC2 instance IDs.

Queries the EC2 API for instance details and writes a YAML inventory
that uses the aws_ssm connection plugin (SSM Session Manager) instead of SSH.

Usage:
    python3 generate_inventory.py \
        --instance-ids "i-0123456789abcdef0,i-0987654321fedcba0" \
        --region us-east-1 \
        --output /playbooks/inventory.yml
"""

import argparse
import sys
import yaml
import boto3
from botocore.exceptions import ClientError


def get_instance_details(instance_ids: list[str], region: str) -> dict[str, dict]:
    """
    Query EC2 for instance details (ID, private IP, state).

    Returns:
        dict mapping instance_id -> { 'private_ip': str, 'state': str }
    """
    ec2 = boto3.client('ec2', region_name=region)

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


def generate_inventory(instances: dict[str, dict], region: str, remote_user: str = 'ec2-user', s3_bucket: str = '') -> dict:
    """
    Build an Ansible inventory structure using the aws_ssm connection plugin.

    SSM Session Manager does not require SSH keys or open inbound ports.
    The target EC2 instances only need the SSM Agent running and an
    instance profile with AmazonSSMManagedInstanceCore policy.
    """
    hosts = {}
    for instance_id, details in instances.items():
        host_vars = {
            'ansible_host': instance_id,  # SSM uses instance ID, not IP
            'ansible_connection': 'community.aws.aws_ssm',
            'ansible_aws_ssm_region': region,
            'ansible_user': remote_user,
        }
        if s3_bucket:
            host_vars['ansible_aws_ssm_bucket_name'] = s3_bucket
        hosts[instance_id] = host_vars

    group_vars = {
        'ansible_connection': 'community.aws.aws_ssm',
        'ansible_aws_ssm_region': region,
        'ansible_user': remote_user,
    }
    if s3_bucket:
        group_vars['ansible_aws_ssm_bucket_name'] = s3_bucket

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
        '--s3-bucket',
        default='',
        help='S3 bucket for SSM file transfers (ansible_aws_ssm_bucket_name)',
    )

    args = parser.parse_args()

    # Parse instance IDs
    instance_ids = [iid.strip() for iid in args.instance_ids.split(',') if iid.strip()]
    if not instance_ids:
        print('[generate_inventory] ERROR: No valid instance IDs provided', file=sys.stderr)
        sys.exit(1)

    print(f'[generate_inventory] Looking up {len(instance_ids)} instance(s) in {args.region}...')

    # Query EC2
    instances = get_instance_details(instance_ids, args.region)

    if not instances:
        print('[generate_inventory] ERROR: No reachable instances found', file=sys.stderr)
        sys.exit(1)

    print(f'[generate_inventory] Found {len(instances)} running instance(s)')

    # Generate and write inventory
    inventory = generate_inventory(instances, args.region, args.remote_user, args.s3_bucket)

    try:
        with open(args.output, 'w') as f:
            yaml.dump(inventory, f, default_flow_style=False)
        print(f'[generate_inventory] Inventory written to {args.output}')
    except OSError as e:
        print(f'[generate_inventory] ERROR: Failed to write inventory to {args.output}: {e}', file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
