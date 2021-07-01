package endpoint

import (
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

// GetUserStorageKeys - get all keys for this user in repo
func GetUserStorageKeys(request *http.Request, linkSvc linkSvc) ([]string, string, error) {
	props, _ := request.Context().Value(ctxKey{}).(jwt.MapClaims)
	//fmt.Println(props["uid"])
	UID := fmt.Sprintf("%v", props["uid"])

	storageKeys, err := linkSvc.List(UID)
	return storageKeys, UID, err
}

// ValidateRequestShortLink - валидация shortlink параметра в request
// Возвращает саму ссылку, юзерайди (из токена), результат - тру - валидно
// инвалидно - результ - фалз, и все пустое.
func ValidateRequestShortLink(request *http.Request, linkSvc linkSvc) (string, string, bool) {

	storageKeys, UID, err := GetUserStorageKeys(request, linkSvc)
	if err != nil {
		return "", "", false
	}

	params := mux.Vars(request)
	shortURL := params["shortlink"]

	for _, storageKey := range storageKeys {
		if storageKey == shortURL {
			return UID, storageKey, true
		}
	}

	return "", "", false
}
