package endpoint_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/app/endpoint"
	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/repository"
)

// crud api test
func TestHandler(t *testing.T) {

	type TokenAnswer struct {
		Access  string `json:"accessToken"`
		Refresh string `json:"refreshToken"`
	}

	//setup server
	// remove file
	os.Remove("test.json")
	var repoif repository.RepoIf
	// подстановка в интерфейс соотвествующего хранилища
	repoif = new(repository.FileRepo)
	// вызов метода интерфейса - инициализация конфигa
	linkSVC := repoif.New("test.json")
	handler := endpoint.RegisterPublicHTTP(linkSVC)

	// auth test /////////////////////////////////////////////////////////////////////////////////////
	// setup test request
	var jsonStr = []byte(`{"uid":"any user"}`)
	req, err := http.NewRequest("POST", "/user/auth", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	// без этого мидлварь не может внутри себя различать запросы, т.к. это делает программа.
	req.RequestURI = "/user/auth"
	// json тип данных
	req.Header.Set("Content-Type", "application/json")

	// ResponseRecorder store api answer for test request
	rr := httptest.NewRecorder()
	// execute server with test request
	handler.ServeHTTP(rr, req)
	// Проверяем статус-код ответа
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	//  работаем с телом ответа
	p, errR := ioutil.ReadAll(rr.Body)
	if errR != nil {
		t.Fail()
	} else {
		if strings.Contains(string(p), "error") {
			t.Errorf("header response shouldn't return error: %s", p)
		} else if !strings.Contains(string(p), `accessToken`) {
			t.Errorf("header response doesn't match:\n%s", p)
		}
	}
	// get accesstoken to go on with further api test
	var jsonTokens = TokenAnswer{}
	err = json.Unmarshal(p, &jsonTokens)
	if err != nil {
		t.Fail()
	}

	fmt.Printf("accessToken %s", jsonTokens.Access)

	// post test create item /////////////////////////////////////////////////////////////////////////////////////

	jsonStr = []byte(`{ "url": "www.mail.ru","shorturl": "abrashabra.cadabra","datetime": "0001-01-01T00:00:00Z","active": 1,"redirs": 1}`)
	req, err = http.NewRequest("POST", "/links", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req.RequestURI = "/links"
	// json тип данных
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jsonTokens.Access)

	rr = httptest.NewRecorder()
	// execute server with test request
	handler.ServeHTTP(rr, req)
	// Проверяем статус-код ответа
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}
	//  работаем с телом ответа
	p, errR = ioutil.ReadAll(rr.Body)
	if errR != nil {
		t.Fail()
	} else {
		if strings.Contains(string(p), "error") {
			t.Errorf("header response shouldn't return error: %s", p)
		} else if !strings.Contains(string(p), `shorturl`) {
			t.Errorf("header response doesn't match:\n%s", p)
		}
	}

	// list test /////////////////////////////////////////////////////////////////////////////////////
	req, err = http.NewRequest("GET", "/links/all", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RequestURI = "/links/all"
	req.Header.Set("Authorization", "Bearer "+jsonTokens.Access)

	rr = httptest.NewRecorder()
	// execute server with test request
	handler.ServeHTTP(rr, req)
	// Проверяем статус-код ответа
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	//  работаем с телом ответа
	p, errR = ioutil.ReadAll(rr.Body)
	if errR != nil {
		t.Fail()
	} else {
		if strings.Contains(string(p), "error") {
			t.Errorf("header response shouldn't return error: %s", p)
		} else if !strings.Contains(string(p), `abrashabra.cadabra`) {
			t.Errorf("header response doesn't match:\n%s", p)
		}
	}

	// shorturl open test ////////////////////////////////////////////////////////////////////////////////////////////
	req, err = http.NewRequest("GET", "/shortopen/abrashabra.cadabra", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RequestURI = "/shortopen/"
	req.Header.Set("Authorization", "Bearer "+jsonTokens.Access)

	rr = httptest.NewRecorder()
	// execute server with test request
	handler.ServeHTTP(rr, req)
	// Проверяем статус-код ответа
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusSeeOther)
	}
	//  работаем с телом ответа
	p, errR = ioutil.ReadAll(rr.Body)
	if errR != nil {
		t.Fail()
	} else {
		if strings.Contains(string(p), "error") {
			t.Errorf("header response shouldn't return error: %s", p)
		} else if !strings.Contains(string(p), `mail.ru`) {
			t.Errorf("header response doesn't match:\n%s", p)
		}
	}
	// update item test /////////////////////////////////////////////////////////////////////////////////////
	jsonStr = []byte(`{ "url": "www.mail.ruUU","shorturl": "abrashabra.cadabra","redirs": 12345}`)
	req, err = http.NewRequest("PUT", "/links/abrashabra.cadabra", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}
	req.RequestURI = "/links/"
	// json тип данных
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jsonTokens.Access)

	rr = httptest.NewRecorder()
	// execute server with test request
	handler.ServeHTTP(rr, req)
	// Проверяем статус-код ответа
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	//  работаем с телом ответа
	p, errR = ioutil.ReadAll(rr.Body)
	if errR != nil {
		t.Fail()
	} else {
		if strings.Contains(string(p), "error") {
			t.Errorf("header response shouldn't return error: %s", p)
		} else if !strings.Contains(string(p), `shorturl`) {
			t.Errorf("header response doesn't match:\n%s", p)
		}
	}

	// delete item test /////////////////////////////////////////////////////////////////////////////////////
	req, err = http.NewRequest("DELETE", "/links/"+"abrashabra.cadabra", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RequestURI = "/links/"
	req.Header.Set("Authorization", "Bearer "+jsonTokens.Access)

	rr = httptest.NewRecorder()
	// execute server with test request
	handler.ServeHTTP(rr, req)
	// Проверяем статус-код ответа
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	//  работаем с телом ответа
	p, errR = ioutil.ReadAll(rr.Body)
	if errR != nil {
		t.Fail()
	} else {
		if strings.Contains(string(p), "error") {
			t.Errorf("header response shouldn't return error: %s", p)
		} else if !strings.Contains(string(p), ``) {
			t.Errorf("header response doesn't match:\n%s", p)
		}
	}

	//api token refresh test /////////////////////////////////////////////////////////////////////////////////////
	req, err = http.NewRequest("POST", "/token/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RequestURI = "/token/refresh"
	// задаем токен рефреш по нему должны получить новую пару токенов
	req.Header.Set("Authorization", "Bearer "+jsonTokens.Refresh)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	rr = httptest.NewRecorder()
	// execute server with test request
	handler.ServeHTTP(rr, req)

	//  работаем с телом ответа
	p, errR = ioutil.ReadAll(rr.Body)
	if errR != nil {
		t.Fail()
	} else {
		if strings.Contains(string(p), "error") {
			t.Errorf("header response shouldn't return error: %s", p)
		} else if !strings.Contains(string(p), `accessToken`) {
			t.Errorf("header response doesn't match:\n%s", p)
		}
	}

	//error token refresh test //////////////////////////////////////////////////////////////////////////////
	req, err = http.NewRequest("POST", "/token/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RequestURI = "/token/refresh"
	// задаем токен рефреш по нему должны получить новую пару токенов
	req.Header.Set("Authorization", "Bearer "+jsonTokens.Access)

	rr = httptest.NewRecorder()
	// execute server with test request
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusUnauthorized)
	}
	//  работаем с телом ответа
	p, errR = ioutil.ReadAll(rr.Body)
	if errR != nil {
		t.Fail()
	} else {
		if !strings.Contains(string(p), `Please provide refresh token`) {
			t.Errorf("header response doesn't match:\n%s", p)
		}
	}

	// remove file
	os.Remove("test.json")
}
