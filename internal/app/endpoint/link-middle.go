package endpoint

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// lint error fix - did not like string type
type ctxKey struct{}

//PromMiddlewareFunc - оценивает время обработки и выводит его в гистограмму /metrics
func PromMiddlewareFunc(promif PromIf) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var prom = promif
			start := time.Now()
			next.ServeHTTP(w, r)
			// Гистограмма апдейт
			// для каждого метода GET, POST etc, будет свое распределение
			endtime := time.Since(start).Microseconds()
			prom.UpdateHist(r.Method, float64(endtime))
		})
	}
}

// LoggingMiddleware - logs any request to api
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		//вызов следующего хендлера в цепочке
		next.ServeHTTP(w, r)
		props, _ := r.Context().Value(ctxKey{}).(jwt.MapClaims)
		UID := fmt.Sprintf("%v", props["uid"])
		log.Printf("request: %s %s, user: %s - %v\n",
			r.Method,
			r.URL.EscapedPath(),
			UID,
			time.Since(start),
		)
		/*
			log.Printf("--> %s %s", r.Method, r.URL.Path)

			lrw := negroni.NewResponseWriter(w)
			next.ServeHTTP(lrw, r)

			statusCode := lrw.Status()
			log.Printf("<-- %d %s", statusCode, http.StatusText(statusCode))
		*/
	})

}

// GenJWTWithClaims - generate jwt tokens pair
func GenJWTWithClaims(uidText string, tokenType int) (string, error) {
	mySigningKey := []byte("AllYourBase")

	type MyCustomClaims struct {
		UID string `json:"uid"`
		jwt.StandardClaims
	}
	// type 0  access token is valid for 24 hours
	var timeExpiry = time.Now().Add(time.Hour * 24).Unix()
	var issuer = "weblink_access"

	if tokenType == 1 {
		// refresh token type 1 is valid for 5 days
		timeExpiry = time.Now().Add(time.Hour * 24 * 5).Unix()
		issuer = "weblink_refresh"
	}

	// Create the Claims
	claims := MyCustomClaims{
		uidText,
		jwt.StandardClaims{
			ExpiresAt: timeExpiry, // access token will expire in 24h after creating
			Issuer:    issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(mySigningKey)
	if err != nil {
		return "", err
	}
	//fmt.Printf("%v %v", ss, err)
	return ss, nil
	//Output: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIiLCJleHAiOjE1MDAwLCJpc3MiOiJ0ZXN0In0.HE7fK0xOQwFEr4WDgRWj4teRPZ6i3GLwD5YCm6Pwu_c <nil>
}

// JWTCheckMiddleware - check for authorization and json flag
func JWTCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.RequestURI == "/user/auth" {
			//bypass jwt check when authenticating
			next.ServeHTTP(w, r)
			return
		}

		if r.RequestURI == "/user/register" {
			//bypass jwt check when authenticating
			next.ServeHTTP(w, r)
			return
		}

		if r.RequestURI == "/metrics" {
			//bypass jwt check when access prom metrics
			next.ServeHTTP(w, r)
			return
		}

		checkif := 1 // db case svc.WhoAmI()
		if checkif == 0 {
			// bypass middle ware token logic in old version using file storage
			re := regexp.MustCompile(`/shortopen/`)
			res := re.FindStringSubmatch(r.RequestURI)
			if len(res) != 0 {
				//bypass jwt check when authenticating
				next.ServeHTTP(w, r)
				return
			}
		}

		authHeader := strings.Split(r.Header.Get("Authorization"), "Bearer ")

		if len(authHeader) != 2 {
			ResponseAPIError(w, 2, http.StatusUnauthorized)
			return
		}

		// get jwtToken
		jwtToken := authHeader[1]
		token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			SECRETKEY := "AllYourBase"
			return []byte(SECRETKEY), nil
		})

		if token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				ctx := context.WithValue(r.Context(), ctxKey{}, claims)

				if r.RequestURI != "/token/refresh" {
					// allow access to all API nodes with access token
					iss := fmt.Sprintf("%v", claims["iss"])
					if iss == "weblink_access" {
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				} else {
					//allow only refresh tokens to go to /token/refresh endpoint
					//check type of token iss should be weblink_refresh
					iss := fmt.Sprintf("%v", claims["iss"])
					if iss == "weblink_refresh" {
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
					ResponseAPIError(w, 7, http.StatusUnauthorized)
					return
				}

			} else {
				log.Printf("%v \n", err)
				ResponseAPIError(w, 2, http.StatusUnauthorized)
				return
			}

		} else if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				log.Printf("Token is either expired or not active yet %v", err)
				ResponseAPIError(w, 1, http.StatusUnauthorized)
				return
			}
		}

		ResponseAPIError(w, 3, http.StatusUnauthorized)
	})
}
