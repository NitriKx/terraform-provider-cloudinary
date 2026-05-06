data "cloudinary_product_environment" "production" {
  id = "abc123def456"
}

output "cloud_name" {
  value = data.cloudinary_product_environment.production.cloud_name
}
