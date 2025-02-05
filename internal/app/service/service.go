package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/repository"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/model"
)

// repo имеет тип интерфейс
// cервисный интерфейс позволяет кешировать хранилище
type cachedrepo interface {
	Get(ctx context.Context, uid, key string, su bool) (model.DataEl, error)
	Put(ctx context.Context, uid, key string, value model.DataEl, su bool) error
	Del(ctx context.Context, uid, key string, su bool) (string, error)
	List(ctx context.Context, uid string) ([]string, error)
	GetUn(ctx context.Context, shortlink string) (string, error)
	PutUser(value model.User) (string, error)
	DelUser(uid string) error
	GetUser(uid string) (model.User, error)
	WhoAmI() uint64
	PayUser(ctx context.Context, uidA, uidB, amount string) error
	FindSuperUser() (string, error)
	GetAll(ctx context.Context, uid string) (model.Data, error)
	AuthUser(user model.User) (string, error)
	GetAllUsers() (model.Users, error)
}

// Service - содержит член repo
type Service struct {
	repo     cachedrepo
	repcache *cache.Cache
}

// New - конструктор Service
func New(repo cachedrepo) *Service {
	// connect to redis server
	rdb := redis.NewClient(&redis.Options{
		Addr: "192.168.1.204:6379", Password: "", // no password set
		DB: 0, // use default DB
	})
	repcache := cache.New(&cache.Options{
		Redis:      rdb,
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})
	log.Printf("redis is connected")
	return &Service{
		repo:     repo,
		repcache: repcache,
	}
}

// New stub method
func (s *Service) New(ctx context.Context, filename string, tracer trace.Tracer) repository.RepoIf {
	panic("implement me")
}

// Get - when get from storage
func (s *Service) Get(ctx context.Context, uid, key string, su bool) (model.DataEl, error) {
	value, err := s.repo.Get(ctx, uid, key, su)
	if err != nil {
		log.Printf("service/Get: get from repo err: %v", err)
		return model.DataEl{}, err
	}
	return value, nil
}

// Put - when put to storage
func (s *Service) Put(ctx context.Context, uid, key string, value model.DataEl, su bool) error {
	if err := s.repo.Put(ctx, uid, key, value, su); err != nil {
		log.Printf("service/Put: put repo err: %v", err)
		return err
	}
	s.flushcacheList(ctx, uid)
	return nil
}

// flushcacheList - when db updated flush cache row related to it
func (s *Service) flushcacheList(ctx context.Context, uid string) {
	key := fmt.Sprintf("uid_LIST:%s", uid)
	if s.repcache.Exists(ctx, key) {
		err := s.repcache.Delete(ctx, key)
		if err != nil {
			log.Printf("service/flushcache: del cache err: %v", err)
		}
		log.Printf("cache of List for %s is deleted", uid)
	}
	s.flushcacheGetAll(ctx, uid)
}

// flushcacheGetAll - when db updated flush cache row related to it
func (s *Service) flushcacheGetAll(ctx context.Context, uid string) {
	key := fmt.Sprintf("uid_GETALL:")
	if s.repcache.Exists(ctx, key) {
		err := s.repcache.Delete(ctx, key)
		if err != nil {
			log.Printf("service/flushcache: del cache err: %v", err)
		}
		log.Printf("cache of GetAll for %s is deleted", uid)
	}
}

// Del when delete from storage
func (s *Service) Del(ctx context.Context, uid, key string, su bool) (string, error) {
	uid, err := s.repo.Del(ctx, uid, key, su)
	if err != nil {
		log.Printf("service/Del: del repo err: %v", err)
		return "", err
	}
	s.flushcacheList(ctx, uid)
	return "", nil
}

// List - when get list of keys from storage
func (s *Service) List(ctx context.Context, uid string) ([]string, error) {
	key := fmt.Sprintf("uid_LIST:%s", uid)
	var items []string
	var dbitems []string
	var err1 error
	err2 := s.repcache.Get(ctx, key, &items)

	if err2 == nil {
		log.Printf("items for %s are from cache", uid)
		return items, nil
	}

	dbitems, err1 = s.repo.List(ctx, uid)
	if err1 != nil {
		log.Printf("service/List: get from repo err: %v", err1)
		return nil, err1
	}

	err := s.repcache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: dbitems,
		TTL:   time.Hour,
	})
	if err != nil {
		log.Printf("items for %s cannot be put to cache: err: %v", uid, err)
	}

	if err2 == cache.ErrCacheMiss {
		log.Printf("items for %s are absent in cache and taken from repo", uid)
		return dbitems, nil
	}

	log.Printf("items for %s are taken from repo because cache redis has problem", uid)
	return dbitems, nil
}

// GetAll - get all links in db (only in pg mode)
func (s *Service) GetAll(ctx context.Context, uid string) (model.Data, error) {

	key := fmt.Sprintf("uid_GETALL:")
	var items model.Data
	var dbitems model.Data
	var err1 error

	err2 := s.repcache.Get(ctx, key, &items)

	if err2 == nil {
		log.Printf("items (getall) for %s are from cache", uid)
		return items, nil
	}

	dbitems, err1 = s.repo.GetAll(ctx, uid)
	if err1 != nil {
		log.Printf("service/GetAll: repo err: %v", err1)
		return model.Data{}, err1
	}

	err := s.repcache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: dbitems,
		TTL:   time.Hour,
	})
	if err != nil {
		log.Printf("items (getall) for %s cannot be put to cache: err: %v", uid, err)
	}

	if err2 == cache.ErrCacheMiss {
		log.Printf("items (getall) for %s are absent in cache and taken from repo", uid)
		return dbitems, nil
	}

	log.Printf("items (getall) for %s are taken from repo because cache redis has problem", uid)
	return dbitems, nil

}

// GetUn - get unique link unanimously from storage
// when open link it increases counter as well and makes payment for opening
func (s *Service) GetUn(ctx context.Context, shortlink string) (string, error) {
	value, err := s.repo.GetUn(ctx, shortlink)
	if err != nil {
		log.Printf("service/GetUn: from repo err: %v", err)
		return "", err
	}
	s.flushcacheGetAll(ctx, "dummy")
	return value, nil
}

// CloseConn - stub method
func (s *Service) CloseConn() {
}

// PutUser - register new or update user
func (s *Service) PutUser(value model.User) (string, error) {
	val, err := s.repo.PutUser(value)
	if err != nil {
		log.Printf("service/PutUser: putuser repo err: %v", err)
		return "", err
	}
	return val, nil
}

// DelUser - when delete user
func (s *Service) DelUser(uid string) error {
	if err := s.repo.DelUser(uid); err != nil {
		log.Printf("service/UserDel: userdel repo err: %v", err)
		return err
	}
	return nil
}

// GetUser - when get user profile
func (s *Service) GetUser(uid string) (model.User, error) {
	value, err := s.repo.GetUser(uid)
	if err != nil {
		log.Printf("service/GetUser: getuser from repo err: %v", err)
		return model.User{}, err
	}
	return value, nil
}

// WhoAmI - when check interface type (file or pg)
func (s *Service) WhoAmI() uint64 {
	return s.repo.WhoAmI()
}

// PayUser - payment one user to another (tx)
func (s *Service) PayUser(ctx context.Context, uidA, uidB, amount string) error {
	if err := s.repo.PayUser(ctx, uidA, uidB, amount); err != nil {
		log.Printf("service/PayUser: payuser repo err: %v", err)
		return err
	}
	return nil
}

// FindSuperUser - find who is su (get suid)
func (s *Service) FindSuperUser() (string, error) {
	value, err := s.repo.FindSuperUser()
	if err != nil {
		log.Printf("service/FindSU: find su repo err: %v", err)
		return "", err
	}
	return value, nil
}

// AuthUser - user login (only in pg mode)
func (s *Service) AuthUser(user model.User) (string, error) {
	value, err := s.repo.AuthUser(user)
	if err != nil {
		log.Printf("service/AuthUser: repo err: %v", err)
		return "", err
	}
	return value, nil
}

// GetAllUsers - get all user profiles (called only by su and pg mode)
func (s *Service) GetAllUsers() (model.Users, error) {
	value, err := s.repo.GetAllUsers()
	if err != nil {
		log.Printf("service/GetAll: find su repo err: %v", err)
		return model.Users{}, err
	}
	return value, nil
}
