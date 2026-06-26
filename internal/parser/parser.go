package parser

import (
	"regexp"
	"strings"
)

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

	//DHL International Express Standard 10-digit footprint
	dhlRegex = regexp.MustCompile(`\b([0-9]{10})\b`)

	// OSM Worldwide: Captures their prefix layout or falls back to USPS handoffs
	osmRegex = regexp.MustCompile(`(?i)\b(OSM[0-9]{10})\b`)

	// Amazon Hub Locker 6-Digit Pickup Token Regex
	amazonLockerRegex = regexp.MustCompile(`(?i)(?:locker|pickup|code|pin)[^\d]*([0-9]{6})\b`)
)

func ParseEmailBody(body string) *LogisticsPayload {
	payload := &LogisticsPayload{}
	normalized := strings.ToLower(body)

	// 1. Check for Amazon Hub Locker Verification Codes
	if matches := amazonLockerRegex.FindStringSubmatch(normalized); len(matches) > 1 {
		payload.LockerCode = matches[1]
		payload.IsLockerToken = true
		payload.Carrier = "Amazon Hub"
		return payload
	}

	// 2. Check for Carrier Footprints
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
		// Quick edge case check: ensure 10-digit DHL numbers don't get misidentified as 12-digit FedEx
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
