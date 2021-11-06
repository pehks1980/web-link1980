package model

import "time"

// Data  - json array for Data
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

// Users - array of user for json
type Users struct {
	Data []User `json:"data"`
}

// User - элемент json
type User struct {
	UID     string `json:"uid"`
	Name    string `json:"name"`
	Passwd  string `json:"passwd"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	Balance string `json:"balance"`
}
