package models

// SeedPage represents a page in the seed data, with its translations.
type SeedPage struct {
	Name           string            `json:"Name"`
	IsCoreSolution bool              `json:"IsCoreSolution"`
	Icon           string            `json:"Icon"`
	Translations   []PageTranslation `json:"Translations"`
}

// SeedSettingTranslation allows for flexible Value types during seeding.
type SeedSettingTranslation struct {
	Key          string      `json:"Key"`
	LanguageCode string      `json:"LanguageCode"`
	Value        interface{} `json:"Value"`
}

// SeedData represents the structure of seed_data.json
type SeedData struct {
	Pages               []SeedPage               `json:"Pages"`
	Settings            []Setting                `json:"Settings"`
	SettingTranslations []SeedSettingTranslation `json:"SettingTranslations"`
}