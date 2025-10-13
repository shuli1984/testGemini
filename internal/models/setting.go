package models

// Setting defines the structure for a key-value setting in the database.
type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}
