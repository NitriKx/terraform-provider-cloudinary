# Using a Cloudinary URL with a separate account URL for provisioning (recommended)
provider "cloudinary" {
  # The cloudinary_url includes media API credentials in the form:
  # cloudinary://API_KEY:API_SECRET@CLOUD_NAME
  # It is recommended to use the CLOUDINARY_URL environment variable instead.
  cloudinary_url = var.cloudinary_url

  # The account_url provides dedicated credentials for the Provisioning API
  # (cloudinary_product_environment, cloudinary_product_environment_access_key,
  # cloudinary_custom_policy). Format: account://ACCOUNT_API_KEY:ACCOUNT_API_SECRET@ACCOUNT_ID
  # It is recommended to use the CLOUDINARY_ACCOUNT_URL environment variable instead.
  account_url = var.cloudinary_account_url
}

# Alternative: explicit credentials
provider "cloudinary" {
  cloud_name  = var.cloudinary_cloud_name
  api_key     = var.cloudinary_api_key
  api_secret  = var.cloudinary_api_secret
  account_url = var.cloudinary_account_url
}
