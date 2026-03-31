package loadbalancer

import (
	"math"
	"net"
	"strings"
)

// GeoIPResolver resolves IP addresses to geographical locations.
type GeoIPResolver struct {
	// In a production implementation, this would use a GeoIP database
	// like MaxMind GeoIP2 or similar
	countries map[string]string // IP prefix -> country code
}

// NewGeoIPResolver creates a new GeoIP resolver.
func NewGeoIPResolver() *GeoIPResolver {
	return &GeoIPResolver{
		countries: loadDefaultGeoData(),
	}
}

// Resolve resolves an IP address to a country code.
func (g *GeoIPResolver) Resolve(ip string) string {
	// Parse IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "UNKNOWN"
	}

	// For IPv4, extract first 2 octets
	if parsedIP.To4() != nil {
		prefix := parsedIP.String()[:strings.LastIndex(parsedIP.String(), ".")]
		prefix = prefix[:strings.LastIndex(prefix, ".")]
		if country, ok := g.countries[prefix]; ok {
			return country
		}
	}

	return "UNKNOWN"
}

// GeoAwareSelector selects targets based on geographic proximity.
type GeoAwareSelector struct {
	resolver *GeoIPResolver
	// Target locations: target ID -> country code
	targetLocations map[string]string
}

// NewGeoAwareSelector creates a new geo-aware selector.
func NewGeoAwareSelector() *GeoAwareSelector {
	return &GeoAwareSelector{
		resolver:        NewGeoIPResolver(),
		targetLocations: make(map[string]string),
	}
}

// SetTargetLocation sets the location for a target.
func (g *GeoAwareSelector) SetTargetLocation(targetID, countryCode string) {
	g.targetLocations[targetID] = countryCode
}

// Select selects the closest target based on client IP.
func (g *GeoAwareSelector) Select(clientIP string, targetIDs []string) string {
	if len(targetIDs) == 0 {
		return ""
	}

	clientCountry := g.resolver.Resolve(clientIP)

	// Find targets in the same country
	for _, id := range targetIDs {
		if location, ok := g.targetLocations[id]; ok && location == clientCountry {
			return id
		}
	}

	// Fall back to first target
	return targetIDs[0]
}

// loadDefaultGeoData loads bundled GeoIP data.
// In production, this would load from a MaxMind database file.
func loadDefaultGeoData() map[string]string {
	// Simplified mapping of some IP ranges to countries
	return map[string]string{
		"192.168": "US",
		"10.0":    "US",
		"172.16":  "EU",
		"127.0":   "LOCAL",
	}
}

// GeoDistanceCalculator calculates distances between geographic coordinates.
type GeoDistanceCalculator struct {
	targetCoords map[string]Coordinates // target ID -> coordinates
}

// Coordinates represents geographic coordinates.
type Coordinates struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
}

// NewGeoDistanceCalculator creates a new distance calculator.
func NewGeoDistanceCalculator() *GeoDistanceCalculator {
	return &GeoDistanceCalculator{
		targetCoords: make(map[string]Coordinates),
	}
}

// SetTargetCoords sets coordinates for a target.
func (g *GeoDistanceCalculator) SetTargetCoords(targetID string, coords Coordinates) {
	g.targetCoords[targetID] = coords
}

// Distance calculates the haversine distance between two coordinates in km.
func (g *GeoDistanceCalculator) Distance(c1, c2 Coordinates) float64 {
	const earthRadius = 6371 // km

	lat1Rad := c1.Lat * math.Pi / 180
	lat2Rad := c2.Lat * math.Pi / 180
	deltaLat := (c2.Lat - c1.Lat) * math.Pi / 180
	deltaLong := (c2.Long - c1.Long) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLong/2)*math.Sin(deltaLong/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// NearestTarget finds the nearest target to the given coordinates.
func (g *GeoDistanceCalculator) NearestTarget(clientCoords Coordinates, targetIDs []string) string {
	if len(targetIDs) == 0 {
		return ""
	}

	var nearestID string
	var minDistance float64

	for _, id := range targetIDs {
		targetCoords, ok := g.targetCoords[id]
		if !ok {
			continue
		}

		dist := g.Distance(clientCoords, targetCoords)
		if nearestID == "" || dist < minDistance {
			nearestID = id
			minDistance = dist
		}
	}

	if nearestID == "" {
		return targetIDs[0] // Fallback
	}

	return nearestID
}

