# VPC for Fargate Tasks - uses public subnets (no NAT Gateway, cost-optimized)

resource "aws_vpc" "terraform_runner" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "terraform-runner-vpc"
    Environment = var.environment
  }
}

# Internet Gateway
resource "aws_internet_gateway" "terraform_runner" {
  vpc_id = aws_vpc.terraform_runner.id

  tags = {
    Name        = "terraform-runner-igw"
    Environment = var.environment
  }
}

# Public Subnets (Fargate tasks run here with public IPs)
resource "aws_subnet" "public" {
  count                   = length(local.availability_zones)
  vpc_id                  = aws_vpc.terraform_runner.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = local.availability_zones[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name        = "terraform-runner-public-${count.index + 1}"
    Environment = var.environment
    Type        = "public"
  }
}

# Route Table for Public Subnets
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.terraform_runner.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.terraform_runner.id
  }

  tags = {
    Name        = "terraform-runner-public-rt"
    Environment = var.environment
  }
}

# Associate public subnets with route table
resource "aws_route_table_association" "public" {
  count          = length(local.availability_zones)
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# Security Group for Fargate Tasks
resource "aws_security_group" "fargate_tasks" {
  name        = "terraform-runner-fargate-sg"
  description = "Security group for Terraform runner Fargate tasks"
  vpc_id      = aws_vpc.terraform_runner.id

  # Allow all outbound traffic (needed for Terraform to access AWS APIs)
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound traffic"
  }

  tags = {
    Name        = "terraform-runner-fargate-sg"
    Environment = var.environment
  }
}
