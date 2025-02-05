package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/repository"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/model"
)

// repo имеет тип интерфейс
// cервисный интерфейс позволяет кешировать хранилище
type cachedwbrepo interface {
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

// ServiceWb - интерфейс кеша с Writeback
type ServiceWb struct {
	repo       cachedwbrepo //repo
	cacheWb    *cache.Cache //основной как бы репозиторий
	workers    []*Worker    // cache workers - ждут Task из канал Qin и делают его что там надо сделать
	Qin        chan *Task
	qbroker    *QBroker // one cache broker - диспетчер очереди Qin - формирует Task и кладет его в Qin
	ctx        context.Context
	cancelFunc context.CancelFunc
	tracer     trace.Tracer
}

// NewWb - конструктор ServiceWb
func NewWb(repo cachedwbrepo, tracer trace.Tracer) *ServiceWb {
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

	//init cache workers
	nWorkers := 2
	Qin := make(chan *Task)
	qbroker := &QBroker{Qin}

	var workers []*Worker
	for i := 0; i < nWorkers; i++ {
		worker := NewWorker(i, Qin)
		workers = append(workers, worker)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	servicewb := &ServiceWb{
		repo:       repo,
		cacheWb:    repcache,
		workers:    workers,
		Qin:        Qin,
		qbroker:    qbroker,
		ctx:        ctx,
		cancelFunc: cancelFunc,
		tracer:     tracer,
	}

	// start workers (to work along with the cache)
	for _, worker := range workers {
		go worker.ProcessQ(ctx, servicewb)
	}

	return servicewb
}

// New stub method
func (s *ServiceWb) New(ctx context.Context, filename string, tracer trace.Tracer) repository.RepoIf {
	panic("implement me")
}

// Get - when get from storage
func (s *ServiceWb) Get(ctx context.Context, uid, key string, su bool) (model.DataEl, error) {
	value, err := s.repo.Get(ctx, uid, key, su)
	if err != nil {
		log.Printf("service/Get: get from repo err: %v", err)
		return model.DataEl{}, err
	}
	return value, nil
}

// Put - when put to storage
// writes to cache, then worker puts it from cache to repo
func (s *ServiceWb) Put(ctx context.Context, uid, key string, value model.DataEl, su bool) error {

	//put item to cache
	cachekey := fmt.Sprintf("uid_PUT:%s", uid)
	err := s.cacheWb.Set(&cache.Item{
		Ctx:   ctx,
		Key:   cachekey,
		Value: value,
		TTL:   time.Hour,
	})

	//span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, s.tracer, "wb.uid_PUT:")
	//defer span.Finish()

	ctx, span := s.tracer.Start(ctx, "wb.uid_PUT:")
	defer span.End()

	if err != nil {
		log.Printf("items (getall) for %s cannot be put to cache: err: %v", uid, err)
		return err
	}
	//make worker put value from cache to database acync
	s.qbroker.ProduceWr(ctx, uid, key, su)
	log.Printf("soon will put data to repo...by some worker (executed ProduceWr) uid=%s ", uid)

	span.AddEvent("wb.uid_PUT:", trace.WithAttributes(
		attribute.String("cache write behind to repo task started", uid),
	))

	key = fmt.Sprintf("uid_LIST:%s", uid)
	s.flushCache(ctx, key)
	s.flushCache(ctx, "uid_GETALL:")
	return nil
}

// flushCache - when db updated flush cache with key uid_LIST or uid_GETALL as key
func (s *ServiceWb) flushCache(ctx context.Context, key string) {
	if s.cacheWb.Exists(ctx, key) {
		err := s.cacheWb.Delete(ctx, key)
		if err != nil {
			log.Printf("service/flushcache: del cache err: %v", err)
		}
		log.Printf("cache for key %s is deleted", key)
	}
}

// Del when delete from storage
func (s *ServiceWb) Del(ctx context.Context, uid, key string, su bool) (string, error) {
	uid, err := s.repo.Del(ctx, uid, key, su)
	if err != nil {
		log.Printf("service/Del: del repo err: %v", err)
		return "", err
	}
	//flush List key
	key = fmt.Sprintf("uid_LIST:%s", uid)
	s.flushCache(ctx, key)
	s.flushCache(ctx, "uid_GETALL:")
	return "", nil
}

// List - when get list of keys
// gets it from cache, when miss it asks worker to get it from repo
// if it is ok it returns the list straight away (so you dont have to repeat reading from cache)
func (s *ServiceWb) List(ctx context.Context, uid string) ([]string, error) {
	key := fmt.Sprintf("uid_LIST:%s", uid)
	var items []string

	//span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, s.tracer, "wb.uid_LIST:")
	//defer span.Finish()
	ctx, span := s.tracer.Start(ctx, "wb.uid_LIST:")
	defer span.End()

	err2 := s.cacheWb.Get(ctx, key, &items)

	if err2 == nil {
		log.Printf("items for %s are from cache", uid)

		span.AddEvent("wb.uid_LIST:", trace.WithAttributes(
			attribute.String("items (uid_LIST) are from cache", uid),
		))
		return items, nil
	}

	items, err := s.qbroker.Produce(ctx, uid, key)
	if err == nil {
		log.Printf("getting data from repo... by some worker (executed Produce) uid=%s \n", uid)
		span.AddEvent("wb.uid_LIST:", trace.WithAttributes(
			attribute.String("items (uid_LIST) are absent in cache and taken from repo", uid),
		))
		return items, nil
	}

	// err is error from worker gorutine
	log.Printf("items for %s cannot work with cache/repo: err: %v", uid, err)

	return nil, err
}

// GetAll - get all links in db (only in pg mode)
func (s *ServiceWb) GetAll(ctx context.Context, uid string) (model.Data, error) {

	key := fmt.Sprintf("uid_GETALL:")
	var items model.Data
	var dbitems model.Data
	var err1 error

	//span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, s.tracer, "wb.uid_GETALL:")
	//defer span.Finish()

	ctx, span := s.tracer.Start(ctx, "wb.uid_GETALL:")
	defer span.End()

	err2 := s.cacheWb.Get(ctx, key, &items)

	if err2 == nil {
		log.Printf("items (getall) for %s are from cache", uid)

		span.AddEvent("wb.uid_GETALL:", trace.WithAttributes(
			attribute.String("items are from cache", uid),
		))

		return items, nil
	}

	dbitems, err1 = s.repo.GetAll(ctx, uid)
	if err1 != nil {
		log.Printf("service/GetAll: repo err: %v", err1)
		return model.Data{}, err1
	}

	err := s.cacheWb.Set(&cache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: dbitems,
		TTL:   time.Hour,
	})
	if err != nil {
		log.Printf("items (uid_GETALL:) for %s cannot be put to cache: err: %v", uid, err)
	}

	if err2 == cache.ErrCacheMiss {
		log.Printf("items (uid_GETALL:) for %s are absent in cache and taken from repo", uid)
		span.AddEvent("wb.uid_GETALL:", trace.WithAttributes(
			attribute.String("items are absent in cache and taken from repo", uid),
		))
		return dbitems, nil
	}

	log.Printf("items (uid_GETALL:) for %s are taken from repo because cache redis has problem", uid)
	return dbitems, nil

}

// GetUn - get unique link unanimously from storage
// when open link it increases counter as well and makes payment for opening
func (s *ServiceWb) GetUn(ctx context.Context, shortlink string) (string, error) {
	value, err := s.repo.GetUn(ctx, shortlink)
	if err != nil {
		log.Printf("service/GetUn: from repo err: %v", err)
		return "", err
	}
	//will remove only cache for uid_GETALL:
	s.flushCache(ctx, "uid_GETALL:")
	return value, nil
}

// CloseConn - stub method
func (s *ServiceWb) CloseConn() {
	s.cancelFunc()
	close(s.Qin)
}

// PutUser - register new or update user
func (s *ServiceWb) PutUser(value model.User) (string, error) {
	val, err := s.repo.PutUser(value)
	if err != nil {
		log.Printf("service/PutUser: putuser repo err: %v", err)
		return "", err
	}
	return val, nil
}

// DelUser - when delete user
func (s *ServiceWb) DelUser(uid string) error {
	if err := s.repo.DelUser(uid); err != nil {
		log.Printf("service/UserDel: userdel repo err: %v", err)
		return err
	}
	return nil
}

// GetUser - when get user profile
func (s *ServiceWb) GetUser(uid string) (model.User, error) {
	value, err := s.repo.GetUser(uid)
	if err != nil {
		log.Printf("service/GetUser: getuser from repo err: %v", err)
		return model.User{}, err
	}
	return value, nil
}

// WhoAmI - when check interface type (file or pg)
func (s *ServiceWb) WhoAmI() uint64 {
	return s.repo.WhoAmI()
}

// PayUser - payment one user to another (tx)
func (s *ServiceWb) PayUser(ctx context.Context, uidA, uidB, amount string) error {
	if err := s.repo.PayUser(ctx, uidA, uidB, amount); err != nil {
		log.Printf("service/PayUser: payuser repo err: %v", err)
		return err
	}
	return nil
}

// FindSuperUser - find who is su (get suid)
func (s *ServiceWb) FindSuperUser() (string, error) {
	value, err := s.repo.FindSuperUser()
	if err != nil {
		log.Printf("service/FindSU: find su repo err: %v", err)
		return "", err
	}
	return value, nil
}

// AuthUser - user login (only in pg mode)
func (s *ServiceWb) AuthUser(user model.User) (string, error) {
	value, err := s.repo.AuthUser(user)
	if err != nil {
		log.Printf("service/AuthUser: repo err: %v", err)
		return "", err
	}
	return value, nil
}

// GetAllUsers - get all user profiles (called only by su and pg mode)
func (s *ServiceWb) GetAllUsers() (model.Users, error) {
	value, err := s.repo.GetAllUsers()
	if err != nil {
		log.Printf("service/GetAll: find su repo err: %v", err)
		return model.Users{}, err
	}
	return value, nil
}
