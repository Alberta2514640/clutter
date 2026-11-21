# The website behind B's load balancer, on the site's one name: B's rule
# (adapter-service/alb.tf, priority 10) takes /v1/* and /healthz, and this rule
# takes everything else for that name.

resource "aws_lb_target_group" "site" {
  name                 = "clutter-website"
  port                 = 3000
  protocol             = "HTTP"
  target_type          = "ip"
  vpc_id               = var.vpc_id
  deregistration_delay = 30

  # The home page is prerendered when the image is built, so the server
  # answers it from memory without calling A or B.
  health_check {
    path                = "/"
    matcher             = "200"
    interval            = 30
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

resource "aws_lb_listener_rule" "site" {
  listener_arn = var.listener_arn
  priority     = var.listener_rule_priority

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.site.arn
  }

  condition {
    host_header {
      values = [var.domain]
    }
  }
}

# --- The website's tasks: accept only the load balancer ---

resource "aws_security_group" "site" {
  name        = "clutter-website"
  description = "Website tasks: only the load balancer may reach them"
  vpc_id      = var.vpc_id
}

resource "aws_vpc_security_group_ingress_rule" "site_from_alb" {
  security_group_id            = aws_security_group.site.id
  description                  = "From the load balancer"
  ip_protocol                  = "tcp"
  from_port                    = 3000
  to_port                      = 3000
  referenced_security_group_id = var.alb_security_group_id
}

resource "aws_vpc_security_group_egress_rule" "site_https" {
  security_group_id = aws_security_group.site.id
  description       = "ECR and CloudWatch Logs, to pull the image and write logs"
  ip_protocol       = "tcp"
  from_port         = 443
  to_port           = 443
  cidr_ipv4         = "0.0.0.0/0"
}

resource "aws_vpc_security_group_egress_rule" "alb_to_site" {
  security_group_id            = var.alb_security_group_id
  description                  = "To the website"
  ip_protocol                  = "tcp"
  from_port                    = 3000
  to_port                      = 3000
  referenced_security_group_id = aws_security_group.site.id
}
