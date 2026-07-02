package parser

import (
	"log"
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
	// Note: Relaxed word boundaries (\b) allow token extraction out of dense HTML markup.
	usps22DigitRegex = regexp.MustCompile(`(9[2345][0-9]{20})`)
	fedexRegex       = regexp.MustCompile(`([0-9]{12}|[0-9]{15})`)
	upsRegex         = regexp.MustCompile(`(?i)(1Z[A-Z0-9]{16})`)
	osmRegex         = regexp.MustCompile(`(?i)(OSM[0-9]{10})`)
	dhlStrictRegex   = regexp.MustCompile(`\b([0-9]{10})\b`)

	// Resilient Amazon Hub Locker 6-Digit Pickup Token Regex
	amazonLockerRegex = regexp.MustCompile(`(?i)Pickup code\s+(\d{6})`)

	// Etsy Link Extraction Regex
	etsyRegex = regexp.MustCompile(`(?i)etsy\.com/shipping/track/([^?\s"\']+)`)
)

// ParseEmailBody scans raw email data for active package tracking numbers or smart locker pickup codes.
func ParseEmailBody(body string) *LogisticsPayload {
	payload := &LogisticsPayload{}
	normalized := strings.ToLower(body)

	// =========================================================================
	// 1. SCAN FOR SMART LOCKER PIN TOKENS
	// =========================================================================
	// Processed via raw body string to preserve and evaluate whitespace line breaks
	if matches := amazonLockerRegex.FindStringSubmatch(body); len(matches) > 1 {
		payload.LockerCode = matches[1]
		payload.IsLockerToken = true
		payload.Carrier = "Amazon Hub"
		return payload
	}

	// =========================================================================
	// 2. INTERCEPT CLOSED-ECOSYSTEM PLATFORM SIGNATURES
	// =========================================================================
	if strings.Contains(normalized, "ablink.account.etsy.com") || (strings.Contains(normalized, "etsy") && strings.Contains(normalized, "track package")) {
		payload.TrackingNumber = "MANUAL_ACTION_REQUIRED"
		payload.Carrier = "Etsy (Action Required)"
		return payload
	}

	if strings.Contains(normalized, "no-reply@shop.app") || strings.Contains(normalized, "track with the shop app") || (strings.Contains(normalized, "shop pay") && strings.Contains(normalized, "shipped")) {
		payload.TrackingNumber = "MANUAL_ACTION_REQUIRED"
		payload.Carrier = "Shop App (Action Required)"
		return payload
	}

	if strings.Contains(normalized, "shipment-tracking@amazon.com") ||
		strings.Contains(normalized, "your amazon.com shipment") ||
		(strings.Contains(normalized, "amazon") && strings.Contains(normalized, "shipped") && !strings.Contains(normalized, "locker")) {
		payload.TrackingNumber = "MANUAL_ACTION_REQUIRED"
		payload.Carrier = "Amazon"
		return payload
	}

	// =========================================================================
	// 3. EXTRACT COLD CARRIER TELEMETRY
	// =========================================================================

	// Scan for clean Etsy Link footprints (non-ablink style)
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

	// High-certainty 22-digit sequence (USPS / OSM Final Mile)
	if usps22DigitRegex.MatchString(body) {
		payload.TrackingNumber = usps22DigitRegex.FindString(body)
		if strings.Contains(normalized, "wizmo") || strings.Contains(normalized, "osm") {
			payload.Carrier = "OSM"
		} else {
			payload.Carrier = "USPS"
		}
		return payload
	}

	// Standard UPS 18-character tracking sequence with checksum integrity verification
	if upsRegex.MatchString(body) {
		extractedUPS := upsRegex.FindString(body)
		if IsValidUPSChecksum(extractedUPS) {
			payload.TrackingNumber = extractedUPS
			payload.Carrier = "UPS"
			return payload
		}
		log.Printf("[SECURITY] Intercepted false-positive UPS ad signature token: %s", extractedUPS)
	}

	// Standard FedEx tracking string matching 12 or 15-digit sequences
	if fedexRegex.MatchString(body) {
		payload.TrackingNumber = fedexRegex.FindString(body)
		payload.Carrier = "FedEx"
		return payload
	}

	// Standard OSM standalone routing sequence
	if osmRegex.MatchString(body) {
		payload.TrackingNumber = osmRegex.FindString(body)
		payload.Carrier = "OSM"
		return payload
	}

	// Standard DHL standalone 10-digit identification token
	if strings.Contains(normalized, "dhl") && dhlStrictRegex.MatchString(body) {
		payload.TrackingNumber = dhlStrictRegex.FindString(body)
		payload.Carrier = "DHL"
		return payload
	}

	return payload
}

// IsValidUPSChecksum implements the standard Modulus 10 validation algorithm
// for 18-character UPS tracking numbers (format: 1Z ... 16 characters).
func IsValidUPSChecksum(tracking string) bool {
	if len(tracking) != 18 {
		return false
	}

	tracking = strings.ToUpper(tracking)
	if tracking[:2] != "1Z" {
		return false
	}

	checkDigitChar := tracking[17]
	sum := 0

	// Loop over the 15 tracking payload characters following the "1Z" prefix
	for i := 2; i < 17; i++ {
		r := tracking[i]
		var val int

		if r >= '0' && r <= '9' {
			val = int(r - '0')
		} else if r >= 'A' && r <= 'Z' {
			// UPS alphanumeric translation lookup map:
			// A-I map to 3-1, J-R map to 2-0, S-Z map to 2-9 (using single digit boundaries)
			val = int(r-'A'+3) % 10
		} else {
			return false
		}

		// Weight alternation scheme: Multiply even-indexed bytes by 2, odd by 1
		if i%2 == 0 {
			sum += val * 2
		} else {
			sum += val
		}
	}

	remainder := sum % 10
	calculatedCheck := 0
	if remainder != 0 {
		calculatedCheck = 10 - remainder
	}

	expectedCheckDigitChar := byte(calculatedCheck + '0')
	return checkDigitChar == expectedCheckDigitChar
}
