// +build integration

package repository_test

// go test --tags=integration . --run tests + integration tests. ,
// '/ +build integration' avoids test to be runned by 'go test .'
import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/model"
	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/repository"
	"github.com/stretchr/testify/require"
)

func TestIntegrationSVCFileRepo(t *testing.T) {
	var repoif, linkSVC repository.RepoIf
	repoif = new(repository.FileRepo)
	linkSVC = repoif.New("test_storage.json")
	// init test struct
	tests := []struct {
		name     string
		alldata  model.DataEl
		keylist  []string
		err      error
		UID      string
		prepare  func() string
		testfunc func(UID string) (model.DataEl, []string, error)
		check    func(t *testing.T, alldata model.DataEl, keylist []string, err error)
		remove   func(UID string)
	}{ // slice
		{ // struct
			name: "test1",
			prepare: func() string {
				fmt.Print("prepare\n")

				// uid test_uid1
				uid := "test_uid1"

				// add some data to db for this uid
				userdata := model.DataEl{
					UID:      uid,
					URL:      "mail.ru",
					Shorturl: "abracadabra.gu",
					Redirs:   100,
					Datetime: time.Now(),
					Active:   1,
				}

				_ = linkSVC.Put(uid, userdata.Shorturl, userdata, false)

				return uid
			},
			testfunc: func(UID string) (model.DataEl, []string, error) {
				fmt.Print("run FileRepo Get \n")
				data, err := linkSVC.Get(UID, "abracadabra.gu", false)
				return data, nil, err
			},
			check: func(t *testing.T, alldata model.DataEl, keylist []string, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, alldata)
			},
			remove: func(UID string) {
				fmt.Print("remove\n")
				linkSVC.Del(UID, "abracadabra.gu", false)
			},
		},
		{ // struct
			name: "test2",
			prepare: func() string {
				fmt.Print("prepare\n")

				// uid test_uid1
				uid := "test_uid1"

				// add some data to db for this uid
				userdata := model.DataEl{
					UID:      uid,
					URL:      "mail.ru",
					Shorturl: "abracadabra.gu",
					Redirs:   100,
					Datetime: time.Now(),
					Active:   1,
				}

				_ = linkSVC.Put(uid, userdata.Shorturl, userdata, false)

				return uid
			},
			testfunc: func(UID string) (model.DataEl, []string, error) {
				fmt.Print("run FileRepo Update (PUT) \n")

				// add some data to db for this uid
				userdata := model.DataEl{
					UID:      UID,
					URL:      "mail.ruuuu",
					Shorturl: "abracadabra.gu",
					Redirs:   100500,
					Datetime: time.Now(),
					Active:   1,
				}

				err := linkSVC.Put(UID, "abracadabra.gu", userdata, false)
				return model.DataEl{}, nil, err
			},
			check: func(t *testing.T, alldata model.DataEl, keylist []string, err error) {
				require.NoError(t, err)
				//require.NotEmpty(t, alldata)
			},
			remove: func(UID string) {
				fmt.Print("remove\n")
				//linkSVC.Del(UID, "abracadabra.gu", false)
			},
		},
		{ // struct
			name: "test3",
			prepare: func() string {
				fmt.Print("prepare\n")

				// uid test_uid1
				uid := "test_uid1"

				// add some data for this uid
				userdata := model.DataEl{
					UID:      uid,
					URL:      "mail.ru",
					Shorturl: "abracadabra.gu",
					Redirs:   100,
					Datetime: time.Now(),
					Active:   1,
				}

				_ = linkSVC.Put(uid, userdata.Shorturl, userdata, false)

				userdata.URL = "yandex.ru"
				userdata.Shorturl = "abrashvabrakadabra.gu"
				userdata.Datetime = time.Now()

				_ = linkSVC.Put(uid, userdata.Shorturl, userdata, false)

				return uid
			},
			testfunc: func(UID string) (model.DataEl, []string, error) {
				fmt.Print("run FileRepo List \n")

				keylist, err := linkSVC.List(UID)
				return model.DataEl{}, keylist, err
			},
			check: func(t *testing.T, alldata model.DataEl, keylist []string, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, keylist)
			},
			remove: func(UID string) {
				fmt.Print("remove\n")
				//linkSVC.Del(UID, "abracadabra.gu", false)
			},
		},
		{ // struct
			name: "test4",
			prepare: func() string {
				fmt.Print("prepare\n")

				// uid test_uid1
				uid := "test_uid1"

				// add some data for this uid
				userdata := model.DataEl{
					UID:      uid,
					URL:      "mail.ru",
					Shorturl: "abracadabra.gu",
					Redirs:   100,
					Datetime: time.Now(),
					Active:   1,
				}

				_ = linkSVC.Put(uid, userdata.Shorturl, userdata, false)

				return uid
			},
			testfunc: func(UID string) (model.DataEl, []string, error) {
				fmt.Print("run FileRepo GetUn \n")

				val, err := linkSVC.GetUn("abracadabra.gu")
				var keylist []string
				// add value to keylist[0]
				keylist = append(keylist, val)
				return model.DataEl{}, keylist, err
			},
			check: func(t *testing.T, alldata model.DataEl, keylist []string, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, keylist)
			},
			remove: func(UID string) {
				fmt.Print("remove\n")
				//linkSVC.Del(UID, "abracadabra.gu", false)
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
			tt.alldata, tt.keylist, tt.err = tt.testfunc(tt.UID)
			tt.check(t, tt.alldata, tt.keylist, tt.err)
			tt.remove(tt.UID)
		})
	}

	// physically remove test json storage file
	_ = os.Remove("test_storage.json")
}
