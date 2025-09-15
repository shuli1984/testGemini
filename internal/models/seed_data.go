package models

// SeedPage represents a page in the seed data, with its translations.
type SeedPage struct {
	Name           string            `json:"Name"`
	IsCoreSolution bool              `json:"IsCoreSolution"`
	Icon           string            `json:"Icon"`
	Translations   []PageTranslation `json:"Translations"`
}

// SeedData represents the structure of seed_data.json
type SeedData struct {
	Pages        []SeedPage    `json:"Pages"`
	SiteSettings []SiteSetting `json:"SiteSettings"`
	MenuItems    []MenuItemDB  `json:"MenuItems"`
}
