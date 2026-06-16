resource "cloudinary_upload_preset" "invoice" {
  name         = "file-uploader-invoice"
  asset_folder = "uaas-v2/file-uploader-preprod-eu/invoice"

  # Prefix the generated public_id with the asset folder so every minted id
  # starts with the folder path (defaults to true).
  use_asset_folder_as_public_id_prefix = true

  resource_type   = "auto"
  unsigned        = false
  type            = "authenticated" # "upload" (public), "authenticated", or "private"
  allowed_formats = ["pdf", "zip"]

  # Server-side snippet: set moderation per file at upload time.
  eval = "var DOC=['pdf','zip'];if(DOC.indexOf((resource_info.format||'').toLowerCase())!=-1){upload_options.moderation='manual';}"
}
