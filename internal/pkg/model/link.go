package model

import "time"

// Data  - json array
type Data struct {
	Data []DataEl `json:"data"`
}

// DataEl - элемент Data строки файла json
type DataEl struct {
	UID      string    `json:"uid"`
	URL      string    `json:"url"`
	Shorturl string    `json:"shorturl"`
	Datetime time.Time `json:"datetime"`
	Active   int       `json:"active"`
	Redirs   int       `json:"redirs"`
}
