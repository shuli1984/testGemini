package models

// SeedData represents the structure of seed_data.json
type SeedData struct {
	Pages        []Page `json:"Pages"`
	SiteSettings []SiteSetting `json:"SiteSettings"`
	MenuItems    []MenuItemDB  `json:"MenuItems"`
}
