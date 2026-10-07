package sharelinks

import "strings"

// How a public open reached the report (health_share_link_views.via).
const (
	ViaLink = "link" // the token URL (a shared link or the scanned QR)
	ViaCode = "code" // the typed 24h code
)

// Coarse client classes of the access log. Nothing finer is kept: no IP, no user agent, no version, no location.
const (
	DeviceMobile  = "mobile"
	DeviceTablet  = "tablet"
	DeviceDesktop = "desktop"
	DeviceUnknown = "unknown"

	BrowserChrome  = "chrome"
	BrowserSafari  = "safari"
	BrowserFirefox = "firefox"
	BrowserEdge    = "edge"
	BrowserSamsung = "samsung"
	BrowserOpera   = "opera"
	BrowserOther   = "other"
)

// Viewer is what the access log records about one public open.
type Viewer struct {
	Via     string
	Device  string
	Browser string
}

// ViewerOf classifies a User-Agent header into a coarse device class and browser family (the header itself is
// dropped).
func ViewerOf(via, userAgent string) Viewer {
	ua := strings.ToLower(userAgent)
	v := Viewer{Via: via, Device: DeviceUnknown, Browser: BrowserOther}
	switch {
	case strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet") ||
		(strings.Contains(ua, "android") && !strings.Contains(ua, "mobile")):
		v.Device = DeviceTablet
	case strings.Contains(ua, "mobi") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android"):
		v.Device = DeviceMobile
	case strings.Contains(ua, "windows") || strings.Contains(ua, "macintosh") || strings.Contains(ua, "x11") ||
		strings.Contains(ua, "linux") || strings.Contains(ua, "cros"):
		v.Device = DeviceDesktop
	}
	switch {
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edga/") || strings.Contains(ua, "edgios/"):
		v.Browser = BrowserEdge
	case strings.Contains(ua, "samsungbrowser"):
		v.Browser = BrowserSamsung
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		v.Browser = BrowserOpera
	case strings.Contains(ua, "firefox/") || strings.Contains(ua, "fxios/"):
		v.Browser = BrowserFirefox
	case strings.Contains(ua, "chrome/") || strings.Contains(ua, "crios/"):
		v.Browser = BrowserChrome
	case strings.Contains(ua, "safari/"):
		v.Browser = BrowserSafari
	}
	return v
}
