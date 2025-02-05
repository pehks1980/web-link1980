package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/cache/v8"
	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/model"
)

// Task - структура элемента очереди для задания worker что делать
// реализованы 2 метода Task1 (cache write behind .List) Task2 (cache write behind .Put)
type Task struct {
	Name   string
	ctx    context.Context
	uid    string
	key    string
	su     bool
	doneCh chan ResultDbItems
}

// Worker - cтруктура воркера
type Worker struct {
	Qin chan *Task
	id  int
	wg  *sync.WaitGroup
}

// NewWorker - Конструктор воркера
func NewWorker(id int, Qin chan *Task) *Worker {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &Worker{
		Qin: Qin,
		id:  id,
		wg:  wg,
	}
}

// ProcessQ - когда по каналу приходит Task он будет обработан горутиной workera
func (w Worker) ProcessQ(ctx context.Context, s *ServiceWb) {
	log.Printf("worker id = %d started.", w.id)
	var resultRead ResultDbItems
	for {
		select {
		case job := <-w.Qin:
			doneCh := job.doneCh
			log.Printf("worker id = %d got task = %s uid=%s", w.id, job.Name, job.uid)

			//TASK 1 LIST method - async read from repo
			if job.Name == "Task1" {
				var dbitems []string
				var err1 error
				dbitems, err1 = s.repo.List(job.ctx, job.uid)
				if err1 != nil {
					log.Printf("service/List: get from repo err: %v", err1)
					resultRead.ResultDb = nil
					resultRead.ResultError = err1
					doneCh <- resultRead
					break
				}

				err := s.cacheWb.Set(&cache.Item{
					Ctx:   job.ctx,
					Key:   job.key,
					Value: dbitems,
					TTL:   time.Hour,
				})

				if err != nil {
					log.Printf("items for %s cannot be put to cache: err: %v", job.uid, err)
					resultRead.ResultDb = nil
					resultRead.ResultError = err1
					doneCh <- resultRead
					break
				}
				log.Printf("worker %d took data for uid = %s from repo to cache successfully \n", w.id, job.uid)
				resultRead.ResultDb = dbitems
				resultRead.ResultError = nil
				doneCh <- resultRead
			}

			//TASK 2 PUT async method push cache to repo
			if job.Name == "Task2" {
				// get data from cache
				cachekey := fmt.Sprintf("uid_PUT:%s", job.uid)
				var value model.DataEl
				err2 := s.cacheWb.Get(job.ctx, cachekey, &value)
				if err2 != nil {
					log.Printf("cannot get value from for %s from cache", job.uid)
					break
					//return items, nil
				}

				if err := s.repo.Put(job.ctx, job.uid, job.key, value, job.su); err != nil {
					log.Printf("service/Put: put repo err: %v", err)
					break
					//same info only !!! )))
				}

				log.Printf("worker %d put data for uid = %s from cache to repo successfully\n", w.id, job.uid)

				//remove item from cache once done with repo
				s.flushCache(ctx, cachekey)

			}

			log.Printf("task = %v finished by %d worker\n", job.Name, w.id)

		case <-ctx.Done():
			w.wg.Done()
			log.Printf("worker id = %d finished.", w.id)
			return
		}
	}
}

// QBroker -  стуктура диспетчера канала управления воркерами
type QBroker struct {
	Qin chan *Task
}

// ResultDbItems - структура для канала данных и ошибок которые возвращает worker
type ResultDbItems struct {
	ResultError error
	ResultDb    []string
}

// Produce - функция генерит задание task1 для исполнения воркером (метод LIST)
func (p QBroker) Produce(ctx context.Context, uid string, key string) ([]string, error) {
	doneCh := make(chan ResultDbItems)
	task := &Task{"Task1",
		ctx,
		uid,
		key,
		false,
		doneCh,
	}
	//put task to channel Queue for worker
	p.Qin <- task

	//wait for worker to retreive data from repo
	res := <-doneCh
	return res.ResultDb, res.ResultError
}

// ProduceWr - функция генерит задание task2 для исполнения воркером (метод PUT)
func (p QBroker) ProduceWr(ctx context.Context, uid string, key string, su bool) {
	doneCh := make(chan ResultDbItems)
	task := &Task{"Task2",
		ctx,
		uid,
		key,
		su,
		doneCh,
	}
	p.Qin <- task
	//просто кидает задание Task2 а воркер его подхватывает и исполняет
}
