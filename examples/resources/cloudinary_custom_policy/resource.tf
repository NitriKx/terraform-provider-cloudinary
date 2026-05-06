# Account-scoped policy
resource "cloudinary_custom_policy" "account_readonly" {
  name        = "account-readonly"
  description = "Read-only access to the entire account"
  scope_type  = "account"
  enabled     = true

  policy_statement = jsonencode({
    effect   = "allow"
    action   = ["read:*"]
    resource = ["*"]
  })
}

# Product environment-scoped policy
resource "cloudinary_product_environment" "staging" {
  name    = "staging"
  enabled = true
}

resource "cloudinary_custom_policy" "staging_readonly" {
  name       = "staging-readonly"
  scope_type = "product_environment"
  scope_id   = cloudinary_product_environment.staging.id
  enabled    = true

  policy_statement = jsonencode({
    effect   = "allow"
    action   = ["read:*"]
    resource = ["*"]
  })
}
