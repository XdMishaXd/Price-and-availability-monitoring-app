package models

import "time"

type Product struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Price        int       `json:"price"`
	In_stock     bool      `json:"in_stock"`
	Last_checked time.Time `json:"last_checked"`
	Created_at   time.Time `json:"created_at"`
	Updated_at   time.Time `json:"updated_at"`
}

type ProductForProducer struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

type ProcutForConsumer struct {
	ID       int64 `json:"id"`
	Price    int   `json:"price"`
	In_stock bool  `json:"in_stock"`
}
