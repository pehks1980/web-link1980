// +build integration

package repository_test

// go test --tags=integration . --run tests + integration tests. ,
// '/ +build integration' avoids test to be runned by 'go test .'
import (
	"fmt"
	"testing"
	"time"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/model"
	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/repository"
	"github.com/stretchr/testify/require"
)

func TestIntegrationSVC(t *testing.T) {
	var repoif, linkSVC repository.RepoIf
	repoif = new(repository.PgRepo)
	linkSVC = repoif.New("postgres://postuser:postpassword@192.168.1.204:5432/a4")
	// init test struct
	tests := []struct {
		name     string
		alldata  model.Data
		user     model.User
		err      error
		UID      []string
		prepare  func() []string
		testfunc func(UID []string) (model.Data, model.User, error)
		check    func(t *testing.T, alldata model.Data, user model.User, err error)
		remove   func(UID []string)
	}{ // slice
		{ // struct
			name: "test1",
			prepare: func() []string {
				fmt.Print("prepare\n")

				user := model.User{
					Name:    "test_user1",
					Passwd:  "123",
					Email:   "L@u.ca",
					Balance: "100.00",
					Role:    "USER",
				}
				var UID []string
				uid, _ := linkSVC.PutUser(user)
				UID = append(UID, uid)
				return UID
			},
			testfunc: func(UID []string) (model.Data, model.User, error) {
				fmt.Print("run GetAll()\n")
				data, err := linkSVC.GetAll()
				return data, model.User{}, err
			},
			check: func(t *testing.T, alldata model.Data, user model.User, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, alldata)
			},
			remove: func(UID []string) {
				fmt.Print("remove\n")
				for _, uid := range UID {
					linkSVC.DelUser(uid)
				}

			},
		},
		{ // struct
			name: "test2",
			prepare: func() []string {
				fmt.Print("prepare\n")

				user := model.User{
					Name:    "test_user1",
					Passwd:  "123",
					Email:   "L@u.ca",
					Balance: "100.00",
					Role:    "USER",
				}
				var UID []string
				uid, _ := linkSVC.PutUser(user)
				UID = append(UID, uid)
				return UID
			},
			testfunc: func(UID []string) (model.Data, model.User, error) {
				fmt.Print("run GetUser\n")
				user, err := linkSVC.GetUser(UID[0])
				return model.Data{}, user, err
			},
			check: func(t *testing.T, alldata model.Data, user model.User, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, user)
			},
			remove: func(UID []string) {
				fmt.Print("remove\n")
				for _, uid := range UID {
					linkSVC.DelUser(uid)
				}

			},
		},
		{ // struct
			name: "test3",
			prepare: func() []string {
				fmt.Print("prepare\n")
				//create test user
				user := model.User{
					Name:    "test_user1",
					Passwd:  "123",
					Email:   "L@u.ca",
					Balance: "100.00",
					Role:    "USER",
				}

				var UID []string
				uid, _ := linkSVC.PutUser(user)
				UID = append(UID, uid)

				// add some data to db
				userdata := model.DataEl{
					URL:      "mail.ru",
					Shorturl: "abracadabra.gu",
					Redirs:   100,
					Datetime: time.Now(),
				}

				_ = linkSVC.Put(uid, userdata.Shorturl, userdata, false)

				return UID
			},
			testfunc: func(UID []string) (model.Data, model.User, error) {
				fmt.Print("run Get\n")
				userdata, err := linkSVC.Get(UID[0], "abracadabra.gu", false)
				var Data model.Data
				Data.Data = append(Data.Data, userdata)
				return Data, model.User{}, err
			},
			check: func(t *testing.T, alldata model.Data, user model.User, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, alldata.Data[0].UID) //some data retrieved
			},
			remove: func(UID []string) {
				fmt.Print("remove\n")
				for _, uid := range UID {
					linkSVC.DelUser(uid)
				}

			},
		},
		{ // struct
			name: "test4",
			prepare: func() []string {
				fmt.Print("prepare\n")
				var UID []string
				//create test user
				user := model.User{
					Name:    "test_user1",
					Passwd:  "123",
					Email:   "L@u.ca",
					Balance: "100.00",
					Role:    "USER",
				}

				uid, _ := linkSVC.PutUser(user)

				UID = append(UID, uid)

				// add some data to db
				userdata := model.DataEl{
					URL:      "mail.ru",
					Shorturl: "abracadabra.gu",
					Redirs:   100,
					Datetime: time.Now(),
				}

				_ = linkSVC.Put(UID[0], userdata.Shorturl, userdata, false)

				user = model.User{
					Name:    "test_user2",
					Passwd:  "123",
					Email:   "L@u.ca",
					Balance: "100.00",
					Role:    "USER",
				}

				uid1, _ := linkSVC.PutUser(user)

				UID = append(UID, uid1)

				return UID
			},
			testfunc: func(UID []string) (model.Data, model.User, error) {
				fmt.Print("run PAY USER 0 to 1\n")
				err := linkSVC.PayUser(UID[0], UID[1], "49.99")
				return model.Data{}, model.User{}, err
			},
			check: func(t *testing.T, alldata model.Data, user model.User, err error) {
				require.NoError(t, err)
			},
			remove: func(UID []string) {
				fmt.Print("remove\n")
				for _, uid := range UID {
					linkSVC.DelUser(uid)
				}

			},
		},
	}

	//run table tests in a cycle
	for _, tt := range tests {
		tt := tt // // capture range variable
		// trick to make sure we pass each test case in this range when we run test in parallel
		// (use t.Parallel()) in run loop
		// though in sequential run it has no use ie https://eleni.blog/2019/05/11/parallel-test-execution-in-go/
		t.Run(tt.name, func(t *testing.T) {
			tt.UID = tt.prepare()
			tt.alldata, tt.user, tt.err = tt.testfunc(tt.UID)
			tt.check(t, tt.alldata, tt.user, tt.err)
			tt.remove(tt.UID)
		})
	}

	defer linkSVC.CloseConn()

}
