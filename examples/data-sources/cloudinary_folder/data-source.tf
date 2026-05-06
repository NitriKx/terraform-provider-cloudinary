data "cloudinary_folder" "assets" {
  path = "production/assets"
}

output "folder_name" {
  value = data.cloudinary_folder.assets.name
}
