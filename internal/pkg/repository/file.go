package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/model"
)

// RepoIf - main methods for a storage (a file repo) same as linkSVC
type RepoIf interface {
	New(filename string, tracer opentracing.Tracer, ctx context.Context) RepoIf
	Get(uid, key string, su bool) (model.DataEl, error)
	Put(uid, key string, value model.DataEl, su bool) error
	Del(uid, key string, su bool) error
	List(uid string, ctx context.Context) ([]string, error)
	GetUn(shortlink string) (string, error)
	CloseConn()
	PutUser(value model.User) (string, error)
	DelUser(uid string) error
	GetUser(uid string) (model.User, error)
	WhoAmI() uint64
	PayUser(uidA, uidB, amount string, ctx context.Context) error
	FindSuperUser() (string, error)
	GetAll() (model.Data, error)
	AuthUser(user model.User) (string, error)
	GetAllUsers() (model.Users, error)
}

//GetAllUsers - stub
func (fr *FileRepo) GetAllUsers() (model.Users, error) {
	return model.Users{}, nil
}

// FileRepo - структура для файло-стораджа
// fileData - мап содержимого файла хешированная as map key := datael.UID + ":" + datael.Shorturl
type FileRepo struct {
	sync.RWMutex
	fileName string
	fileData map[string]model.DataEl
}

//AuthUser - stub
func (fr *FileRepo) AuthUser(user model.User) (string, error) {
	return "", nil
}

// WhoAmI - identification of interface
func (fr *FileRepo) WhoAmI() uint64 {
	return 0
}

// New - инициализация файлостораджа
func (fr *FileRepo) New(filename string, tracer opentracing.Tracer, ctx context.Context) RepoIf {
	// init file repo
	fileRepo := &FileRepo{
		fileName: filename,
		fileData: make(map[string]model.DataEl),
	}
	//check if file exists
	// if yes load from disk and populate repo structs
	// so 'Image' of file is held in map and it gets flushed every time change occurs
	if _, err := os.Stat(filename); err == nil {
		// path/to/whatever exists
		err = fileRepo.FileRepoUnpackToStruct()
		if err != nil {
			log.Fatalf("Problem with filesystem: %v", err)
		}
	}

	return fileRepo
}

// DumpMapToFile - no lock, as its has been done in upper level
func (fr *FileRepo) DumpMapToFile() error {
	// to do dump map to file.json
	// make slice of active links and write it to file
	var fileDataSlice model.Data

	for _, value := range fr.fileData {
		// stripe all not Active when dumping
		if value.Active == 1 {
			fileDataSlice.Data = append(fileDataSlice.Data, value)
		}
	}

	filedata, _ := json.MarshalIndent(fileDataSlice, "", " ")

	err := ioutil.WriteFile(fr.fileName, filedata, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

// FileRepoUnpackToStruct - load file to map of structs
func (fr *FileRepo) FileRepoUnpackToStruct() error {
	fr.RWMutex.Lock() // rw lock while reading file to fr map structure
	defer fr.RWMutex.Unlock()
	// по ссылке извлекаем строку файлового хранилища
	// читаем все в мапу и делаем поиск
	//jsonFile, err := os.Open(fr.fileName)
	jsonFile, err := os.OpenFile(fr.fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("json repo file %s open error", fr.fileName)
		return err
	}

	// Не забываем закрыть файл при выходе из функции
	defer func() {
		var ferr = jsonFile.Close()
		if ferr != nil {
			log.Printf("can't close file: %v", ferr)
		}
	}()

	// read our opened jsonFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)
	// we initialize our data array
	var fileDataSlice model.Data
	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'fileDataSlice' which we defined above
	err = json.Unmarshal(byteValue, &fileDataSlice)
	if err != nil {
		return err
	}
	// quickly populate our file map

	// we iterate through array and make map key [UID:shortlink]=filedata struct
	for _, datael := range fileDataSlice.Data {
		key := datael.UID + ":" + datael.Shorturl
		fr.fileData[key] = datael
	}

	return nil
}

// Get - get data string from repo
// uid:key - user:key
func (fr *FileRepo) Get(uid, key string, su bool) (model.DataEl, error) {
	fr.RWMutex.RLock() // read lock only
	defer fr.RWMutex.RUnlock()
	// get data needed
	// retrieve dat string
	key = uid + ":" + key
	if datael, ok := fr.fileData[key]; ok {
		if datael.Active == 0 {
			// deleted already
			err := fmt.Errorf("link deleted already")
			return model.DataEl{}, err
		}

		return datael, nil

	}
	err := fmt.Errorf("No such link")
	return model.DataEl{}, err
}

// GetUn - find unique shortlink in storage for shortopen api method
// + update redir count (protected by lock)
func (fr *FileRepo) GetUn(shortlink string) (string, error) {
	fr.RWMutex.Lock()
	defer fr.RWMutex.Unlock()
	// get data needed
	// retrieve dat string
	for key, datael := range fr.fileData {
		//strip user: from key
		//check if we have match
		keys := strings.Split(key, ":")
		if keys[1] == shortlink {
			//found unique link
			if datael.Active == 0 {
				// deleted already
				err := fmt.Errorf("link deleted already")
				return "", err
			}
			// update redirs count save it and return it
			datael.Redirs++
			fr.fileData[key] = datael
			// changes needs to be flushed to file
			err := fr.DumpMapToFile()
			if err != nil {
				return "", err
			}
			return datael.URL, nil
		}
	}

	err := fmt.Errorf("No such link")
	return "", err
}

// Put - store data string to repo
func (fr *FileRepo) Put(uid, key string, value model.DataEl, su bool) error {
	fr.RWMutex.Lock()
	defer fr.RWMutex.Unlock()
	/*	if _, ok := fr.fileData[key]; !ok {
		// key already exists
		err := fmt.Errorf("link %s dont exist", key)
		return err
	}*/
	key = uid + ":" + key

	fr.fileData[key] = value
	// changes needs to be flushed to file
	err := fr.DumpMapToFile()
	if err != nil {
		return err
	}
	return nil
}

// Del - mark Active = 0 to 'delete'
func (fr *FileRepo) Del(uid, key string, su bool) error {
	fr.RWMutex.Lock()
	defer fr.RWMutex.Unlock()
	key = uid + ":" + key
	if datael, ok := fr.fileData[key]; ok {
		datael.Active = 0
		fr.fileData[key] = datael
		// dump data to file straight away
		err := fr.DumpMapToFile()
		if err != nil {
			return err
		}
		return nil
	}
	err := fmt.Errorf("delete error key %s don't exist", key)
	return err
}

// List - list all keys for this user uid
func (fr *FileRepo) List(uid string, ctx context.Context) ([]string, error) {
	fr.RWMutex.RLock()
	defer fr.RWMutex.RUnlock()
	// get data needed
	// retrieve list of keys as []string
	var keys []string
	for _, val := range fr.fileData {
		if val.Active == 1 && val.UID == uid {
			keys = append(keys, val.Shorturl)
		}
	}
	return keys, nil
}

// GetAll заглушки
func (fr *FileRepo) GetAll() (model.Data, error) {
	return model.Data{}, nil
}

// PayUser заглушки
func (fr *FileRepo) PayUser(uidA, uidB, amount string, ctx context.Context) error {
	return nil
}

// FindSuperUser заглушки
func (fr *FileRepo) FindSuperUser() (string, error) {
	return "", nil
}

// PutUser заглушки
func (fr *FileRepo) PutUser(value model.User) (string, error) {
	return "", nil
}

// DelUser заглушки
func (fr *FileRepo) DelUser(uid string) error {
	return nil
}

// GetUser заглушки
func (fr *FileRepo) GetUser(uid string) (model.User, error) {
	return model.User{}, nil
}

// AddUser заглушки
func (fr *FileRepo) AddUser(value model.User) (string, error) {
	return "", nil
}

// CloseConn заглушки
func (fr *FileRepo) CloseConn() {

}
