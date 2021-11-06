package endpoint

import (
	"encoding/json"
	"net/http"
)

// Errors - json error api
type Errors struct {
	Errors []ErrorEl `json:"errors"`
}

// ErrorEl - json array
type ErrorEl struct {
	Code    uint64 `json:"code"`
	Message string `json:"message"`
}

var (
	apiErrorList = map[uint64]string{
		1:   "Token has expired time",
		2:   "Unknown token",
		3:   "Access/Refresh token is lost",
		4:   "The shortlink name must be provided.",
		5:   "The shortlink with the specified name already exists",
		6:   "The shortlink with the specified name does not exist",
		7:   "Please provide refresh token, or authenticate again",
		8:   "No uid (user id), please set uid",
		9:   "Unknown content type",
		10:  "Internal repo problem",
		11:  "No shorturl in data",
		12:  "Login error, provide username password",
		400: "Bad request",
		401: "Unauthorized",
		402: "Payment required",
		403: "Forbidden",
		404: "Not found",
		405: "Method not allowed",
	}
)

// ResponseAPIError - reply to api when some error happened when accessing api
func ResponseAPIError(w http.ResponseWriter, code uint64, status int) {
	var errorsjson = Errors{}
	errorel := ErrorEl{Code: code, Message: apiErrorList[code]}
	errorsjson.Errors = append(errorsjson.Errors, errorel)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status) //has to be called first!!!!
	_ = json.NewEncoder(w).Encode(errorsjson)

}
