resource "cloudinary_product_environment" "staging" {
  name    = "staging"
  enabled = true

  custom_attributes = {
    team        = "platform"
    environment = "staging"
  }
}
