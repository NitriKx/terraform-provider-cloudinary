# Look up by ID.
data "cloudinary_trigger" "by_id" {
  id = "abc123"
}

# Look up by event_type + uri.
data "cloudinary_trigger" "by_event" {
  event_type = "upload"
  uri        = "https://example.com/webhooks/cloudinary"
}

output "trigger_auth_scheme" {
  value = data.cloudinary_trigger.by_event.auth_scheme
}
