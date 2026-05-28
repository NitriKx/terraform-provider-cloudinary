data "cloudinary_current_principal" "me" {}

output "current_principal_id" {
  value = data.cloudinary_current_principal.me.principal_id
}

output "current_principal_type" {
  value = data.cloudinary_current_principal.me.principal_type
}
