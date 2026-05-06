resource "cloudinary_folder" "assets" {
  path = "production/assets"
}

resource "cloudinary_folder" "nested" {
  path = "production/assets/images"
}
