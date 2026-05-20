// Package providerdata defines the shared data type passed from the provider
// to all resources and data sources via Configure().
package providerdata

import cloudinary "github.com/cloudinary/cloudinary-go/v2"

// ProviderData carries the Cloudinary client together with the credentials
// that were used to configure it. Resources and data sources receive a
// *ProviderData via their Configure() call.
type ProviderData struct {
	Client    *cloudinary.Cloudinary
	APIKey    string
	CloudName string
}
