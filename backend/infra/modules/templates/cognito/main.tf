resource "aws_cognito_user_pool" "clutter-user-pool" {
  name = "clutter-user-pool"
  auto_verified_attributes = ["email"]
}

resource "aws_cognito_user_pool_domain" "clutter-domain" {
  domain       = "clutter-user"
  user_pool_id = aws_cognito_user_pool.clutter-user-pool.id
}

resource "aws_cognito_identity_provider" "google" {
  user_pool_id  = aws_cognito_user_pool.clutter-user-pool.id
  provider_name = "Google"
  provider_type = "Google"

  provider_details = {
    client_id       = var.google_client_id
    client_secret   = var.google_client_secret
    authorize_scopes = "openid email profile"
  }

  attribute_mapping = {
    email = "email"
  }
  
  lifecycle {
    ignore_changes = [
      attribute_mapping["username"],
      provider_details,
    ]
  }
}

# resource "aws_cognito_identity_provider" "apple" {
#   user_pool_id  = aws_cognito_user_pool.main.id
#   provider_name = "SignInWithApple"
#   provider_type = "SignInWithApple"

#   provider_details = {
#     client_id        = var.apple_client_id
#     team_id          = var.apple_team_id
#     key_id           = var.apple_key_id
#     private_key      = var.apple_private_key
#     authorize_scopes = "email name"
#   }

#   attribute_mapping = {
#     email = "email"
#   }
# }

resource "aws_cognito_user_pool_client" "client" {
  name         = "clutter-frontend-client"
  user_pool_id = aws_cognito_user_pool.clutter-user-pool.id

  allowed_oauth_flows = ["code"]
  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_scopes = ["openid", "email", "profile"]

  callback_urls = [
    "${var.frontend_url}/auth/callback"
  ]

  logout_urls = [
    "${var.frontend_url}/logout"
  ]

  supported_identity_providers = [
    "COGNITO",
    "Google",
    # "SignInWithApple"
  ]
}
