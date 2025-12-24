package models

import "time"

type Product struct {
	ID           int64
	Title        string
	Price        int
	In_stock     bool
	Last_checked time.Time
	Created_at   time.Time
	Updated_at   time.Time
}
