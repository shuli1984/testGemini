package models

// SettingTranslation defines the structure for a translated setting value.
type SettingTranslation struct {
	Key          string `gorm:"primaryKey"`
	LanguageCode string `gorm:"primaryKey"`
	Value        string `gorm:"type:text"`
}
