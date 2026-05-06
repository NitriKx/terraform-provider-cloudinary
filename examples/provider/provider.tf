# Using a Cloudinary URL (simplest method)
provider "cloudinary" {
  # The cloudinary_url includes credentials in the form:
  # cloudinary://API_KEY:API_SECRET@CLOUD_NAME
  # It is recommended to use the CLOUDINARY_URL environment variable instead.
  cloudinary_url = var.cloudinary_url

  # Required for Provisioning API resources (cloudinary_product_environment,
  # cloudinary_product_environment_access_key, cloudinary_custom_policy).
  account_id = var.cloudinary_account_id
}

# Alternative: explicit credentials
provider "cloudinary" {
  cloud_name = var.cloudinary_cloud_name
  api_key    = var.cloudinary_api_key
  api_secret = var.cloudinary_api_secret
  account_id = var.cloudinary_account_id
}
