# Minimal: fire on every upload event.
resource "cloudinary_trigger" "on_upload" {
  uri        = "https://example.com/webhooks/cloudinary"
  event_type = "upload"
}

# Filtered: only fire for image uploads, using a JSONLogic expression.
resource "cloudinary_trigger" "on_image_upload" {
  uri        = "https://example.com/webhooks/cloudinary-images"
  event_type = "upload"
  filter     = jsonencode({ "==" = [{ "var" = "resource_type" }, "image"] })
}

# Custom payload template (Mustache) with EdDSA v2 signature verification.
resource "cloudinary_trigger" "on_delete_custom" {
  uri        = "https://example.com/webhooks/cloudinary-delete"
  event_type = "delete"
  auth_scheme = "eddsa_v2"

  payload_template = jsonencode({
    event = {
      type      = "{{event.type}}"
      timestamp = "{{event.timestamp}}"
    }
    asset = {
      public_id = "{{asset.public_id}}"
      url       = "{{asset.url}}"
    }
  })
}
