# The platform's one load balancer, on the site's one name: HTTPS with one ACM
# certificate for that name, HTTP redirected. The HTTPS listener routes by
# path: /v1/* and /healthz to B API (here), everything else for the site's
# name to the website (modules/website), and any other name to a 404. The
# website has no route under /v1 or /healthz, so neither is shadowed.

resource "aws_lb" "adapter" {
  name                       = "clutter-adapter"
  load_balancer_type         = "application"
  internal                   = false
  security_groups            = [aws_security_group.alb.id]
  subnets                    = aws_subnet.public[*].id
  drop_invalid_header_fields = true
}

resource "aws_lb_target_group" "api" {
  name                 = "clutter-adapter-api"
  port                 = 8080
  protocol             = "HTTP"
  target_type          = "ip"
  vpc_id               = aws_vpc.adapter.id
  deregistration_delay = 30

  health_check {
    path                = "/healthz"
    matcher             = "200"
    interval            = 15
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.adapter.arn
  port              = 443
  protocol          = "HTTPS"
  # The console's default policy (TLS 1.3 and 1.2); the API's own default is
  # the older ELBSecurityPolicy-2016-08.
  ssl_policy      = "ELBSecurityPolicy-TLS13-1-2-Res-PQ-2025-09"
  certificate_arn = var.certificate_arn

  # A request for the load balancer's own address, or any name but the
  # site's, reaches nothing.
  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "text/plain"
      message_body = "Not found"
      status_code  = "404"
    }
  }
}

# Ahead of the website's rule (priority 20), which takes the rest of the name.
resource "aws_lb_listener_rule" "api" {
  listener_arn = aws_lb_listener.https.arn
  priority     = 10

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api.arn
  }

  condition {
    host_header {
      values = [var.domain]
    }
  }

  # Every B route is under /v1/, plus its health check (backend/adapter/internal/api).
  condition {
    path_pattern {
      values = ["/v1/*", "/healthz"]
    }
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.adapter.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "redirect"
    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}
