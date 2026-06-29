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
	usps22DigitRegex = regexp.MustCompile(`(9[2345][0-9]{20})`)
	fedexRegex       = regexp.MustCompile(`([0-9]{12}|[0-9]{15})`)
	upsRegex         = regexp.MustCompile(`(?i)(1Z[A-Z0-9]{16})`)
	osmRegex         = regexp.MustCompile(`(?i)(OSM[0-9]{10})`)

	// A strict 10-digit scanner for real DHL tracking numbers
	dhlStrictRegex = regexp.MustCompile(`\b([0-9]{10})\b`)

	// Resilient Amazon Hub Locker 6-Digit Pickup Token Regex
	amazonLockerRegex = regexp.MustCompile(`(?i)(?:locker|pickup|code|pin)\b.*?([0-9]{6})\b`)

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

	// 2. Intercept encrypted Etsy redirect links
	if strings.Contains(normalized, "ablink.account.etsy.com") || (strings.Contains(normalized, "etsy") && strings.Contains(normalized, "track package")) {
		payload.TrackingNumber = "MANUAL_ACTION_REQUIRED"
		payload.Carrier = "Etsy (Action Required)"
		return payload
	}

	// 3. Intercept closed-ecosystem Shop Pay emails
	if strings.Contains(normalized, "no-reply@shop.app") || strings.Contains(normalized, "track with the shop app") || (strings.Contains(normalized, "shop pay") && strings.Contains(normalized, "shipped")) {
		payload.TrackingNumber = "MANUAL_ACTION_REQUIRED"
		payload.Carrier = "Shop App (Action Required)"
		return payload
	}

	// 4. INTERCEPT AMAZON LOGISTICS EMAIL FOOTPRINTS
	// Bypasses closed-ecosystem tracking number parsing and returns a clean dashboard placeholder.
	if strings.Contains(normalized, "shipment-tracking@amazon.com") ||
		strings.Contains(normalized, "your amazon.com shipment") ||
		(strings.Contains(normalized, "amazon") && strings.Contains(normalized, "shipped")) {

		payload.TrackingNumber = "MANUAL_ACTION_REQUIRED"
		payload.Carrier = "Amazon"
		return payload
	}

	// 5. Scan for clean Etsy Link footprints (non-ablink style)
	if matches := etsyRegex.FindStringSubmatch(body); len(matches) > 1 {
		extractedNum := matches[1]
		payload.TrackingNumber = extractedNum

		if usps22DigitRegex.MatchString(extractedNum) {
			payload.Carrier = "USPS (via Etsy)"
		} else if strings.HasPrefix(strings.ToUpper(extractedNum), "1Z") || upsRegex.MatchString(extractedNum) {
			payload.Carrier = "UPS (via Etsy)"
		} else {
			payload.Carrier = "Etsy Shipping"
		}
		return payload
	}

	// 6. Scan for high-certainty 22-digit sequence (USPS / OSM / Wizmo final mile)
	if usps22DigitRegex.MatchString(body) {
		payload.TrackingNumber = usps22DigitRegex.FindString(body)
		if strings.Contains(normalized, "wizmo") || strings.Contains(normalized, "osm") {
			payload.Carrier = "OSM"
		} else {
			payload.Carrier = "USPS"
		}
		return payload
	}

	// 7. Scan for other standard carrier footprints
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
	if osmRegex.MatchString(body) {
		payload.TrackingNumber = osmRegex.FindString(body)
		payload.Carrier = "OSM"
		return payload
	}

	// 8. Catch DHL strictly if the context says 'dhl' and it finds a standalone 10-digit ID
	if strings.Contains(normalized, "dhl") && dhlStrictRegex.MatchString(body) {
		payload.TrackingNumber = dhlStrictRegex.FindString(body)
		payload.Carrier = "DHL"
		return payload
	}

	return payload
}
