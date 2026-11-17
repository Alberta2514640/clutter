package generic

import "os"

// PlatformRegion is the region Clutter itself runs in (world 1). Lambda and A2
// always set AWS_REGION; the fallback is the cloud environment's region (ADR-015)
// and only matters in tests.
func PlatformRegion() string {
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	return "ca-central-1"
}

// DefaultRegion is the region a user's deploy uses when they choose none: the
// region Clutter runs in.
func DefaultRegion() string {
	return PlatformRegion()
}

var SupportedRegions = map[string]struct{}{
	// United States
	"us-east-1": {}, // N. Virginia
	"us-east-2": {}, // Ohio
	"us-west-1": {}, // N. California
	"us-west-2": {}, // Oregon

	// Asia Pacific
	"ap-south-1":     {}, // Mumbai
	"ap-northeast-3": {}, // Osaka
	"ap-northeast-2": {}, // Seoul
	"ap-southeast-1": {}, // Singapore
	"ap-southeast-2": {}, // Sydney
	"ap-northeast-1": {}, // Tokyo

	// Canada
	"ca-central-1": {}, // Central

	// Europe
	"eu-central-1": {}, // Frankfurt
	"eu-west-1":    {}, // Ireland
	"eu-west-2":    {}, // London
	"eu-west-3":    {}, // Paris
	"eu-north-1":   {}, // Stockholm

	// South America
	"sa-east-1": {}, // São Paulo
}

func IsRegionSupported(region string) bool {
	_, ok := SupportedRegions[region]
	return ok
}
