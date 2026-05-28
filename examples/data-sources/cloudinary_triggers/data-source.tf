data "cloudinary_triggers" "all" {}

output "trigger_count" {
  value = length(data.cloudinary_triggers.all.triggers)
}

output "trigger_uris" {
  value = [for t in data.cloudinary_triggers.all.triggers : t.uri]
}
