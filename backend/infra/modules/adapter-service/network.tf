# B's own VPC: its load balancer, its API and its run tasks, apart from A's
# default-VPC tasks. Public subnets without NAT; tasks get public addresses.
# Nothing in this VPC accepts traffic from a run task: the API accepts only
# the load balancer, run tasks accept nothing, and run tasks sit in their own
# subnets whose network ACL refuses every private address.

data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc" "adapter" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = { Name = "clutter-adapter" }
}

resource "aws_internet_gateway" "adapter" {
  vpc_id = aws_vpc.adapter.id

  tags = { Name = "clutter-adapter" }
}

resource "aws_subnet" "public" {
  count             = 2
  vpc_id            = aws_vpc.adapter.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = { Name = "clutter-adapter-public-${count.index}" }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.adapter.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.adapter.id
  }

  tags = { Name = "clutter-adapter-public" }
}

resource "aws_route_table_association" "public" {
  count          = 2
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# --- Load balancer: HTTPS from anywhere, forwards to B's API and the website ---

resource "aws_security_group" "alb" {
  name        = "clutter-adapter-alb"
  description = "B load balancer"
  vpc_id      = aws_vpc.adapter.id
}

resource "aws_vpc_security_group_ingress_rule" "alb_https" {
  security_group_id = aws_security_group.alb.id
  description       = "HTTPS from browsers"
  ip_protocol       = "tcp"
  from_port         = 443
  to_port           = 443
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_ingress_rule" "alb_http" {
  security_group_id = aws_security_group.alb.id
  description       = "HTTP, redirected to HTTPS"
  ip_protocol       = "tcp"
  from_port         = 80
  to_port           = 80
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_egress_rule" "alb_to_api" {
  security_group_id            = aws_security_group.alb.id
  description                  = "To B API"
  ip_protocol                  = "tcp"
  from_port                    = 8080
  to_port                      = 8080
  referenced_security_group_id = aws_security_group.api.id
}

# --- B API: accepts the load balancer; reaches Postgres and AWS endpoints ---

resource "aws_security_group" "api" {
  name        = "clutter-adapter-api"
  description = "B API tasks"
  vpc_id      = aws_vpc.adapter.id
}

resource "aws_vpc_security_group_ingress_rule" "api_from_alb" {
  security_group_id            = aws_security_group.api.id
  description                  = "From the load balancer"
  ip_protocol                  = "tcp"
  from_port                    = 8080
  to_port                      = 8080
  referenced_security_group_id = aws_security_group.alb.id
}

resource "aws_vpc_security_group_egress_rule" "api_https" {
  security_group_id = aws_security_group.api.id
  description       = "AWS APIs (ECS, S3, STS, Secrets Manager, ECR, CloudWatch Logs) and OpenStack identity endpoints"
  ip_protocol       = "tcp"
  from_port         = 443
  to_port           = 443
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_egress_rule" "api_postgres" {
  for_each          = toset(["5432", "6543"])
  security_group_id = aws_security_group.api.id
  description       = "Postgres (Supabase direct and pooled ports)"
  ip_protocol       = "tcp"
  from_port         = tonumber(each.value)
  to_port           = tonumber(each.value)
  cidr_ipv4         = "0.0.0.0/0"
}

# --- Run tasks: accept nothing; reach the internet for providers and clouds ---

# Run tasks get their own subnets, so the network ACL below applies to them
# and to nothing else in the VPC. A run connects to addresses a customer
# chose (an OpenStack auth URL, the endpoints its catalog lists), and DNS can
# point a public name at a private address after any check: the ACL stops a
# run from reaching this VPC, the load balancer or any other private,
# shared or link-local address. A run needs nothing inside the VPC: it
# reaches S3, ECR, CloudWatch Logs, STS, the Terraform registry and the
# target clouds through the internet gateway, and the VPC's DNS resolver is
# never filtered by an ACL. An ACL also cannot filter the ECS task metadata
# address (169.254.170.2), which holds no credential for a run task: the
# task has no role.
resource "aws_subnet" "run" {
  count             = 2
  vpc_id            = aws_vpc.adapter.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, count.index + 2)
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = { Name = "clutter-adapter-run-${count.index}" }
}

resource "aws_route_table_association" "run" {
  count          = 2
  subnet_id      = aws_subnet.run[count.index].id
  route_table_id = aws_route_table.public.id
}

resource "aws_network_acl" "run" {
  vpc_id     = aws_vpc.adapter.id
  subnet_ids = aws_subnet.run[*].id

  tags = { Name = "clutter-adapter-run" }
}

locals {
  # Private (RFC 1918), shared (RFC 6598), link-local and "this network"
  # ranges, and their IPv6 counterparts. This VPC's own range is inside
  # 10.0.0.0/8 by default; it is named on its own in case it is not.
  run_denied_ipv4 = distinct(["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10", "169.254.0.0/16", "0.0.0.0/8", var.vpc_cidr])
  run_denied_ipv6 = ["fc00::/7", "fe80::/10"]
}

# Stateless: each direction has its own rules. The lowest number wins, so the
# denies (100 and up) come before the final allow (1000).
resource "aws_network_acl_rule" "run_deny_ipv4" {
  for_each = { for i, cidr in local.run_denied_ipv4 : cidr => i }

  network_acl_id = aws_network_acl.run.id
  egress         = true
  rule_number    = 100 + each.value
  rule_action    = "deny"
  protocol       = "-1"
  cidr_block     = each.key
}

resource "aws_network_acl_rule" "run_deny_ipv6" {
  for_each = { for i, cidr in local.run_denied_ipv6 : cidr => i }

  network_acl_id  = aws_network_acl.run.id
  egress          = true
  rule_number     = 200 + each.value
  rule_action     = "deny"
  protocol        = "-1"
  ipv6_cidr_block = each.key
}

resource "aws_network_acl_rule" "run_out" {
  network_acl_id = aws_network_acl.run.id
  egress         = true
  rule_number    = 1000
  rule_action    = "allow"
  protocol       = "-1"
  cidr_block     = "0.0.0.0/0"
}

# Inbound: nothing from a private range (the run's security group accepts no
# connection anyway), and the internet's answers to the run's own requests.
resource "aws_network_acl_rule" "run_in_deny_ipv4" {
  for_each = { for i, cidr in local.run_denied_ipv4 : cidr => i }

  network_acl_id = aws_network_acl.run.id
  egress         = false
  rule_number    = 100 + each.value
  rule_action    = "deny"
  protocol       = "-1"
  cidr_block     = each.key
}

resource "aws_network_acl_rule" "run_in" {
  network_acl_id = aws_network_acl.run.id
  egress         = false
  rule_number    = 1000
  rule_action    = "allow"
  protocol       = "-1"
  cidr_block     = "0.0.0.0/0"
}

resource "aws_security_group" "run" {
  name        = "clutter-adapter-run"
  description = "B run tasks: no inbound traffic"
  vpc_id      = aws_vpc.adapter.id
}

resource "aws_vpc_security_group_egress_rule" "run_tcp" {
  security_group_id = aws_security_group.run.id
  description       = "Terraform registry, provider downloads, S3 and the target clouds' APIs on any port"
  ip_protocol       = "tcp"
  from_port         = 0
  to_port           = 65535
  cidr_ipv4         = "0.0.0.0/0"
}
