resource "cloudinary_product_environment" "staging" {
  name    = "staging"
  enabled = true
}

resource "cloudinary_product_environment_access_key" "api" {
  product_environment_id = cloudinary_product_environment.staging.id
  name                   = "terraform-managed"
  enabled                = true
}

output "api_key" {
  description = "The Cloudinary API key for the staging environment."
  value       = cloudinary_product_environment_access_key.api.api_key
}

output "api_secret" {
  description = "The Cloudinary API secret for the staging environment."
  value       = cloudinary_product_environment_access_key.api.api_secret
  sensitive   = true
}
