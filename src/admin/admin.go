// Package admin contains admin panel rendering and handler helpers.
// See AI.md PART 17 for admin panel specifications.
// Admin panel is completely isolated from the public site.
package admin

// ValidAdminRootPaths defines the only valid first-level paths under /{admin_path}/
// See AI.md PART 17 for route hierarchy rules.
var ValidAdminRootPaths = map[string]bool{
	"":              true,
	"profile":       true,
	"preferences":   true,
	"notifications": true,
	"server":        true,
}

// ReservedPaths cannot be used as admin_path values
var ReservedPaths = []string{
	"api", "static", "assets", "health", "healthz", "version",
	"metrics", ".well-known", "auth", "ws", "openapi", "graphql",
	"dl", "manifest.json",
}

// IsReservedPath checks if a path is reserved and cannot be used as admin_path
func IsReservedPath(path string) bool {
	for _, reserved := range ReservedPaths {
		if path == reserved {
			return true
		}
	}
	return false
}

// SidebarSection represents a group in the admin sidebar
type SidebarSection struct {
	Name  string
	Icon  string
	Items []SidebarItem
}

// SidebarItem represents a single item in the admin sidebar
type SidebarItem struct {
	Name string
	Path string
}

// GetSidebarSections returns the admin panel sidebar navigation
func GetSidebarSections(adminPath string) []SidebarSection {
	return []SidebarSection{
		{
			Name: "Dashboard",
			Icon: "dashboard",
			Items: []SidebarItem{
				{Name: "Overview", Path: "/" + adminPath + "/"},
			},
		},
		{
			Name: "Server",
			Icon: "server",
			Items: []SidebarItem{
				{Name: "Settings", Path: "/" + adminPath + "/server/settings"},
				{Name: "Downloads", Path: "/" + adminPath + "/server/downloads"},
				{Name: "SSL/TLS", Path: "/" + adminPath + "/server/ssl"},
				{Name: "Scheduler", Path: "/" + adminPath + "/server/scheduler"},
				{Name: "Email", Path: "/" + adminPath + "/server/email"},
				{Name: "Logs", Path: "/" + adminPath + "/server/logs"},
				{Name: "Backup", Path: "/" + adminPath + "/server/backup"},
				{Name: "Updates", Path: "/" + adminPath + "/server/updates"},
				{Name: "Info", Path: "/" + adminPath + "/server/info"},
			},
		},
		{
			Name: "Security",
			Icon: "security",
			Items: []SidebarItem{
				{Name: "Authentication", Path: "/" + adminPath + "/server/security/auth"},
				{Name: "API Tokens", Path: "/" + adminPath + "/server/security/tokens"},
				{Name: "Rate Limiting", Path: "/" + adminPath + "/server/security/rate-limit"},
				{Name: "Firewall", Path: "/" + adminPath + "/server/security/firewall"},
			},
		},
		{
			Name: "Network",
			Icon: "network",
			Items: []SidebarItem{
				{Name: "Tor", Path: "/" + adminPath + "/server/network/tor"},
				{Name: "GeoIP", Path: "/" + adminPath + "/server/network/geoip"},
				{Name: "Proxy", Path: "/" + adminPath + "/server/network/proxy"},
			},
		},
		{
			Name: "Media",
			Icon: "media",
			Items: []SidebarItem{
				{Name: "Watch Rules", Path: "/" + adminPath + "/server/media/watch-rules"},
				{Name: "Presets", Path: "/" + adminPath + "/server/media/presets"},
				{Name: "Collections", Path: "/" + adminPath + "/server/media/collections"},
				{Name: "Storage", Path: "/" + adminPath + "/server/media/storage"},
			},
		},
	}
}
