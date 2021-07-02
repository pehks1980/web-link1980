package endpoint

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/model"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

//  linkSvc - интерфейс сервиса со стороны http
//  также имеет put get del crud - для работы с файлохранилищем
//  list - list all links for uid user
//  GetUn - open link for redir and add 1 to redir count
type linkSvc interface {
	Get(uid, key string) (model.DataEl, error)
	Put(uid, key string, value model.DataEl) error
	Del(uid, key string) error
	List(uid string) ([]string, error)
	GetUn(shortlink string) (model.DataEl, error)
}

// RegisterPublicHTTP - регистрация роутинга путей типа urls.py для обработки сервером
func RegisterPublicHTTP(linkSvc linkSvc) *mux.Router {
	// mux golrilla почему он? не знаю, - прикольное название, простота работы..
	r := mux.NewRouter()
	// JWT authorization
	r.HandleFunc("/user/auth", postAuth(linkSvc)).Methods(http.MethodPost)
	r.HandleFunc("/token/refresh", postTokenRefresh(linkSvc)).Methods(http.MethodPost)
	// Main function
	r.HandleFunc("/shortopen/{shortlink}", getShortOpen(linkSvc)).Methods(http.MethodGet)
	r.HandleFunc("/shortstat/{shortlink}", getShortStat(linkSvc)).Methods(http.MethodGet)
	// Links crud
	r.HandleFunc("/links", postToLink(linkSvc)).Methods(http.MethodPost)
	r.HandleFunc("/links/all", getFromLink(linkSvc)).Methods(http.MethodGet)
	r.HandleFunc("/links/{shortlink}", putToLink(linkSvc)).Methods(http.MethodPut)
	r.HandleFunc("/links/{shortlink}", delFromLink(linkSvc)).Methods(http.MethodDelete)
	// MiddleWare first goes JWT second goes Logging
	r.Use(JWTCheckMiddleware)
	r.Use(LoggingMiddleware)
	return r
}

// postTokenRefresh - get new pair of jwt tokens when access token is expired
func postTokenRefresh(svc linkSvc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, request *http.Request) {
		type TokenAnswer struct {
			Access  string `json:"accessToken"`
			Refresh string `json:"refreshToken"`
		}

		props, _ := request.Context().Value(ctxKey{}).(jwt.MapClaims)

		UID := fmt.Sprintf("%v", props["uid"])

		/*
			Issuer := fmt.Sprintf("%v", props["iss"])

			if Issuer != "weblink_refresh" {
				ResponseAPIError(w, 7, http.StatusBadRequest)
				return
			}
		*/
		tokenAccess, _ := GenJWTWithClaims(UID, 0)
		tokenRefresh, _ := GenJWTWithClaims(UID, 1)

		var jsonTokens = TokenAnswer{
			Access:  tokenAccess,
			Refresh: tokenRefresh,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(jsonTokens)

		if err != nil {
			return
		}

	}
}

// postAuth - autheticate and give authorization token
func postAuth(svc linkSvc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, request *http.Request) {
		//json header check
		contentType := request.Header.Get("Content-Type")
		if contentType != "application/json" {
			ResponseAPIError(w, 9, http.StatusBadRequest)
			return
		}

		type TokenAnswer struct {
			Access  string `json:"accessToken"`
			Refresh string `json:"refreshToken"`
		}

		type PostJSONRq struct {
			UID string `json:"uid"`
		}

		var jsonPostRq = PostJSONRq{}

		err := json.NewDecoder(request.Body).Decode(&jsonPostRq)
		if err != nil {
			ResponseAPIError(w, 400, http.StatusBadRequest)
			return
		}
		// get uid
		if jsonPostRq.UID == "" {
			ResponseAPIError(w, 8, http.StatusBadRequest)
			return
		}

		tokenAccess, _ := GenJWTWithClaims(jsonPostRq.UID, 0)
		tokenRefresh, _ := GenJWTWithClaims(jsonPostRq.UID, 1)

		var jsonTokens = TokenAnswer{
			Access:  tokenAccess,
			Refresh: tokenRefresh,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(jsonTokens)
		if err != nil {
			return
		}

		return

	}
}

// delFromLink deletes link from api storage by shortlink
func delFromLink(linkSvc linkSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {

		UID, storageKey, res := ValidateRequestShortLink(request, linkSvc)
		if !res {
			ResponseAPIError(w, 4, http.StatusBadRequest)
			return
		}

		//found key, delete it
		err := linkSvc.Del(UID, storageKey)
		if err != nil {
			ResponseAPIError(w, 10, http.StatusBadRequest)
			return
		}

	}
}

// putToLink updates link from api storage
func putToLink(linkSvc linkSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		//json header check
		contentType := request.Header.Get("Content-Type")
		if contentType != "application/json" {
			ResponseAPIError(w, 9, http.StatusBadRequest)
			return
		}

		var element = model.DataEl{}
		w.Header().Set("Content-Type", "application/json")

		UID, _, res := ValidateRequestShortLink(request, linkSvc)
		if !res {
			ResponseAPIError(w, 9, http.StatusBadRequest)
			return
		}
		//found key, work with body
		err := json.NewDecoder(request.Body).Decode(&element)
		if err != nil {
			ResponseAPIError(w, 9, http.StatusBadRequest)
			return
		}
		element.Datetime = time.Now()
		element.UID = UID
		element.Active = 1
		//looks ok, update storage
		err = linkSvc.Put(UID, element.Shorturl, element)
		if err != nil {
			ResponseAPIError(w, 9, http.StatusBadRequest)
		}
		// form answer json
		err = json.NewEncoder(w).Encode(element)
		if err != nil {
			return
		}
		return
	}
}

// postToLink - creates new item in api storage
func postToLink(linkSvc linkSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		//json header check
		contentType := request.Header.Get("Content-Type")
		if contentType != "application/json" {
			ResponseAPIError(w, 9, http.StatusBadRequest)
			return
		}
		var element = model.DataEl{}
		w.Header().Set("Content-Type", "application/json")

		storageKeys, UID, err := GetUserStorageKeys(request, linkSvc)
		if err != nil {
			ResponseAPIError(w, 10, http.StatusBadRequest)
			return
		}

		err = json.NewDecoder(request.Body).Decode(&element)
		if err != nil {
			ResponseAPIError(w, 9, http.StatusBadRequest)
			return
		}
		// check if we have key
		if element.Shorturl == "" {
			ResponseAPIError(w, 11, http.StatusBadRequest)
			return
		}

		element.Datetime = time.Now()
		// check if this key already exists
		for _, storageKey := range storageKeys {
			if storageKey == element.Shorturl {
				ResponseAPIError(w, 5, http.StatusBadRequest)
				return
			}
		}
		element.UID = UID
		element.Active = 1
		err = linkSvc.Put(UID, element.Shorturl, element)
		if err != nil {
			ResponseAPIError(w, 10, http.StatusBadRequest)
		}
		w.WriteHeader(http.StatusCreated) // this has to be the first write!!!
		err = json.NewEncoder(w).Encode(element)
		if err != nil {
			return
		}
		return

	}
}

// getFromLink - get links list in json
func getFromLink(linkSvc linkSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var datajson = model.Data{}

		storageKeys, UID, err := GetUserStorageKeys(request, linkSvc)
		if err != nil {
			ResponseAPIError(w, 10, http.StatusBadRequest)
			return
		}

		for _, storageKey := range storageKeys {
			getElement, errfor := linkSvc.Get(UID, storageKey)
			if errfor != nil {
				ResponseAPIError(w, 10, http.StatusBadRequest)
				return
				//http.Error(w, "Cannot read from repo", http.StatusBadRequest)
			}

			datajson.Data = append(datajson.Data, getElement)
		}
		// sort by date asc
		sort.Slice(datajson.Data, func(i, j int) bool {
			return datajson.Data[i].Datetime.Before(datajson.Data[j].Datetime)
		})

		err = json.NewEncoder(w).Encode(datajson)
		if err != nil {
			return
		}

	}
}

// getShortStat - get one link from api
func getShortStat(linkSvc linkSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var datajson = model.Data{}
		// check user authorization, get user UID, get key (for this user, check if key exists)
		// if res - yes then do the action  - give string from repo as json
		UID, storageKey, res := ValidateRequestShortLink(request, linkSvc)
		if !res {
			ResponseAPIError(w, 11, http.StatusBadRequest)
			return
		}

		getElement, err := linkSvc.Get(UID, storageKey)
		if err != nil {
			ResponseAPIError(w, 10, http.StatusBadRequest)
			return
		}

		datajson.Data = append(datajson.Data, getElement)
		err = json.NewEncoder(w).Encode(datajson)
		if err != nil {
			return
		}
	}
}

// getShortOpen - get link opened (unonimously)
func getShortOpen(linkSvc linkSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		// get data
		// update data
		// redir to real link

		params := mux.Vars(request)
		shortURL := params["shortlink"]
		// GetUn retreives link and updates redir count
		getElement, err := linkSvc.GetUn(shortURL)
		if err != nil {
			ResponseAPIError(w, 10, http.StatusBadRequest)
			return
		}

		log.Printf("opening user %s link  %s (short is %s) redirs(++) %d \n", getElement.UID, getElement.URL, getElement.Shorturl, getElement.Redirs)
		http.Redirect(w, request, getElement.URL, http.StatusSeeOther)
		//<a href="/shortopen/www.mail.ru">See Other</a>.
		return

	}
}
