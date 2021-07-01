package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/pehks1980/go_gb_be1_kurs/web-link/internal/app/config"
	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/app/endpoint"
	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/repository"
	// репозиторий (хранилище) 1 файло 2 память 3 pg sql(db)
)

// главная петля
func main() {
	log.Print("Starting the app")
	// настройка порта, настроек хранилища, таймаут при закрытии сервиса
	// port := flag.String("port", "8000", "Port")
	storageName := flag.String("storage", "storage.json", "data storage")
	shutdownTimeout := flag.Int64("shutdown_timeout", 3, "shutdown timeout")
/*
	// for heroku env variable PORT (supersedes flag cmd setting)
	basepath, err := os.Getwd()
	if err != nil {
		log.Fatalf("path error %v ", err)
	}
	// load config
	c, errc := config.New(basepath + "/.env")
	if errc != nil {
		log.Fatalf("config error : %v", err)
		return
	}
	//reassign port val from .env file
	port = &c.PORT
*/
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}
	// инициализация файлового хранилища ук на структуру repo
	var repoif repository.RepoIf
	// подстановка в интерфейс соотвествующего хранилища
	repoif = new(repository.FileRepo)
	//repoif = new(repository.MemRepo)
	//repoif = new(repository.PgRepo)

	// вызов метода интерфейса - инициализация конфигa
	linkSVC := repoif.New(*storageName)

	//linkSVC := service.New(repoif) - интерфейс обертка можно использовать для тестинга/мокинга/логинга
	// других сервис-функций

	// repoif <-> linkSVC

	// создание сервера с таким портом, и обработчиком интерфейс которого связывается а файлохранилищем
	// т.к. инициализация происходит (RegisterPublicHTTP)- в интерфейс endpoint подается структура из file.go
	serv := http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: endpoint.RegisterPublicHTTP(linkSVC),
	}
	// запуск сервера
	go func() {
		if err := serv.ListenAndServe(); err != nil {
			log.Fatalf("listen and serve err: %v", err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	log.Printf("Started app at port = %s", port)
	// ждет сигнала
	sig := <-interrupt

	log.Printf("Sig: %v, stopping app", sig)
	// шат даун по контексту с тайм аутом
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*shutdownTimeout)*time.Second)
	defer cancel()
	if err := serv.Shutdown(ctx); err != nil {
		log.Printf("shutdown err: %v", err)
	}
}
