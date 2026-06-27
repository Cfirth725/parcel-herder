package parser

import (
	"regexp"
	"strings"
)

// LogisticsPayload holds the extracted carrier telemetry and locker status tokens.
type LogisticsPayload struct {
	TrackingNumber string
	Carrier        string
	LockerCode     string
	IsLockerToken  bool
}

var (
	// Standard Logistics Carriers Regex
	uspsRegex  = regexp.MustCompile(`\b(94[0-9]{20}|92[0-9]{20})\b`)
	fedexRegex = regexp.MustCompile(`\b([0-9]{12}|[0-9]{15})\b`)
	upsRegex   = regexp.MustCompile(`\b(1Z[A-Z0-9]{16})\b`)
	dhlRegex   = regexp.MustCompile(`\b([0-9]{10})\b`)
	osmRegex   = regexp.MustCompile(`(?i)\b(OSM[0-9]{10})\b`)

	// Amazon Hub Locker 6-Digit Pickup Token Regex
	amazonLockerRegex = regexp.MustCompile(`(?i)(?:locker|pickup|code|pin)[^\d]*([0-9]{6})\b`)

	// Etsy Link Extraction Regex: Matches tracking info inside their URL layout
	etsyRegex = regexp.MustCompile(`(?i)etsy\.com/shipping/track/([^?\s"\']+)`)
)

// ParseEmailBody scans raw email data for active package tracking numbers or smart locker pickup codes.
func ParseEmailBody(body string) *LogisticsPayload {
	payload := &LogisticsPayload{}
	normalized := strings.ToLower(body)

	// 1. Scan for Amazon Hub Locker Verification Codes
	if matches := amazonLockerRegex.FindStringSubmatch(normalized); len(matches) > 1 {
		payload.LockerCode = matches[1]
		payload.IsLockerToken = true
		payload.Carrier = "Amazon Hub"
		return payload
	}

	// 2. Scan for Etsy Link footprints before standard carrier checks
	if matches := etsyRegex.FindStringSubmatch(body); len(matches) > 1 {
		extractedNum := matches[1]
		payload.TrackingNumber = extractedNum

		// Determine the underlying carrier by passing the extracted number back through our rules
		if upsRegex.MatchString(extractedNum) {
			payload.Carrier = "UPS (via Etsy)"
		} else if uspsRegex.MatchString(extractedNum) {
			payload.Carrier = "USPS (via Etsy)"
		} else {
			payload.Carrier = "Etsy Shipping"
		}
		return payload
	}

	// 3. Scan for Carrier Footprints sequentially
	if osmRegex.MatchString(body) {
		payload.TrackingNumber = osmRegex.FindString(body)
		payload.Carrier = "OSM Worldwide"
		return payload
	}
	if uspsRegex.MatchString(body) {
		payload.TrackingNumber = uspsRegex.FindString(body)
		payload.Carrier = "USPS"
		return payload
	}
	if upsRegex.MatchString(body) {
		payload.TrackingNumber = upsRegex.FindString(body)
		payload.Carrier = "UPS"
		return payload
	}
	if fedexRegex.MatchString(body) {
		payload.TrackingNumber = fedexRegex.FindString(body)
		payload.Carrier = "FedEx"
		return payload
	}
	if dhlRegex.MatchString(body) {
		payload.TrackingNumber = dhlRegex.FindString(body)
		payload.Carrier = "DHL"
		return payload
	}

	return payload
}
