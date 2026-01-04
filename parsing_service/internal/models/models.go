package models

type Marketplace string

const (
	Etsy       Marketplace = "etsy"
	Ebay       Marketplace = "ebay"
	Aliexpress Marketplace = "aliexpress"
)

type ProductForProducer struct {
	ID          int64       `json:"id"`
	URL         string      `json:"url"`
	Marketplace Marketplace `json:"marketplace"`
}

type ParsedProduct struct {
	ID       int64 `json:"id"`
	Price    int   `json:"price"`
	In_stock bool  `json:"in_stock"`
}
