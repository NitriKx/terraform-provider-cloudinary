# Changelog

## [0.2.0](https://github.com/NitriKx/terraform-provider-cloudinary/compare/v0.1.1...v0.2.0) (2026-06-16)


### Features

* add cloudinary_upload_preset resource ([#13](https://github.com/NitriKx/terraform-provider-cloudinary/issues/13)) ([33a5736](https://github.com/NitriKx/terraform-provider-cloudinary/commit/33a5736084d74cbca5669f56751b9071ae6eddb4))

## 0.1.1 (2026-05-28)


### Features

* initial Cloudinary Terraform provider ([efa59fa](https://github.com/NitriKx/terraform-provider-cloudinary/commit/efa59fab0a28c2d00a9bae647cdca1389d27cf5d))


### Bug Fixes

* set version to 0.1.0 and restore go release type for release-please ([495c723](https://github.com/NitriKx/terraform-provider-cloudinary/commit/495c723318506c975e9c1c95b89dc06e38450005))

## 0.1.0 (2026-05-28)


### Features

* **provider:** add cloudinary_current_principal data source ([f6cb270](https://github.com/NitriKx/terraform-provider-cloudinary/commit/f6cb270544101fa76e53cecdb15c40faa5d99e4a))
* **provider:** add cloudinary_trigger resource and data sources ([ebe9902](https://github.com/NitriKx/terraform-provider-cloudinary/commit/ebe990294a61bdb2d48bd0c3ccdf497182a46ebf))
* **provider:** initial provider implementation ([753f9f4](https://github.com/NitriKx/terraform-provider-cloudinary/commit/753f9f42a182386aad6043c8ebdfcbf9a8baeb77))


### Bug Fixes

* **provider:** HCL credentials take precedence over CLOUDINARY_URL env var ([cfb1e8d](https://github.com/NitriKx/terraform-provider-cloudinary/commit/cfb1e8d069203110b7bf9eb0dbdfa144931f681a))
* **trigger:** handle not-found via list+filter, fix empty-ID check in Read ([e1e72a9](https://github.com/NitriKx/terraform-provider-cloudinary/commit/e1e72a9366198366faf5bb40973d349361dcf999))
