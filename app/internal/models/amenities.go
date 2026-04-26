package models

// AmenityIcon represents the predefined set of SVG icon names used for
// highlighted amenities. Values must match the icon names in SvgIcon.vue.
type AmenityIcon string

const (
	IconBed        AmenityIcon = "bed"
	IconSnowflake  AmenityIcon = "snowflake" // Air conditioning
	IconTV         AmenityIcon = "tv"
	IconCoffee     AmenityIcon = "coffee"
	IconSofa       AmenityIcon = "sofa"
	IconUtensils   AmenityIcon = "utensils"
	IconHeadphones AmenityIcon = "headphones"
	IconSparkle    AmenityIcon = "sparkle"
	IconBriefcase  AmenityIcon = "briefcase"
	IconMountain   AmenityIcon = "mountain"
	IconLock       AmenityIcon = "lock"
	IconInfo       AmenityIcon = "info"
	IconWifi       AmenityIcon = "wifi"
)

// IconMultipliers maps each icon to its weight in the recommendation coefficient.
// Higher multiplier = stronger positive signal for recommendations.
// Change values here to tune the recommendation algorithm.
var IconMultipliers = map[AmenityIcon]float64{
	IconBed:        1.5,
	IconSnowflake:  1.2,
	IconTV:         1.1,
	IconCoffee:     1.1,
	IconSofa:       1.2,
	IconUtensils:   1.3,
	IconHeadphones: 1.1,
	IconSparkle:    1.4,
	IconBriefcase:  1.1,
	IconMountain:   1.3,
	IconLock:       1.2,
	IconInfo:       1.0,
	IconWifi:       1.5,
}

// CategoryTier represents the quality/importance tier of an amenity category.
// This drives the per-category multiplier in the coef formula.
type CategoryTier string

const (
	TierBasic     CategoryTier = "basic"
	TierEssential CategoryTier = "essential"
	TierComfort   CategoryTier = "comfort"
	TierLuxury    CategoryTier = "luxury"
)

// TierMultipliers maps each tier to its per-amenity weight.
// Formula contribution: TierMultiplier[tier] * category.AmenityCount
// Change values here to tune category weighting.
var TierMultipliers = map[CategoryTier]float64{
	TierBasic:     0.8,
	TierEssential: 1.0,
	TierComfort:   1.2,
	TierLuxury:    1.5,
}
