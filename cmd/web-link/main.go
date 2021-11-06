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

	_ "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	_ "go.uber.org/zap"

	jaegerlog "github.com/uber/jaeger-client-go/log"
	// репозиторий (хранилище) 1 файло 2 память 3 pg sql(db)
)

// главная петля
func main() {
	log.Print("Starting the app")
	// настройка порта, настроек хранилища, таймаут при закрытии сервиса
	portdef := flag.String("port", "8000", "Port")

	storageType := flag.String("storage type", "pg", "data storage type: 'file' or 'pg'")

	storageName := flag.String("storage name", "postgres://postuser:postpassword@192.168.1.204:5432/a4",
		"pg: 'postgres://dbuser:dbpasswd@ip_address:port/dbname'  file: 'storage.json'")

	//storageName := flag.String("storage name", "storage.json",
	//	"pg: 'postgres://dbuser:dbpasswd@ip_address:port/dbname'  file: 'storage.json'")

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
		log.Printf("$PORT is not set. using default %s", *portdef)
		port = *portdef
	}
	// инициализация файлового хранилища ук на структуру repo
	var repoif, linkSVC repository.RepoIf

	// init tracer
	jLogger := jaegerlog.StdLogger
	// tracer config init
	cfg := &config.Configuration{
		ServiceName: "weblink",
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LocalAgentHostPort: "127.0.0.1:6831",
			LogSpans: true,
		},
	}
	jTracer, jCloser, err := cfg.NewTracer(config.Logger(jLogger))

	if err != nil {
		log.Fatalf("cannot init Jaeger err: %v", err)
	}
	// close the closer
	defer jCloser.Close()
	// create empty context for this app
	ctx := context.Background()
	// подстановка в интерфейс соотвествующего хранилища
	if *storageType == "file" {
		repoif = new(repository.FileRepo)
		linkSVC = repoif.New(*storageName, jTracer, ctx)
	}
	if *storageType == "pg" {
		repoif = new(repository.PgRepo)
		linkSVC = repoif.New(*storageName, jTracer, ctx)
		defer linkSVC.CloseConn()

	}

	//repoif = new(repository.MemRepo)

	// repoif <-> linkSVC

	// создание сервера с таким портом, и обработчиком интерфейс которого связывается а файлохранилищем
	// т.к. инициализация происходит (RegisterPublicHTTP)- в интерфейс endpoint подается структура из file.go

	// Prometheus init //////////////////////////////////
	// создаем структуру-интерфейс для прометиуса, включающую 2 обьекта cчетчик и гистограммка
	var promif, Prometh endpoint.PromIf

	promif = new(endpoint.Prom)
	Prometh = promif.New()

	serv := http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: endpoint.RegisterPublicHTTP(linkSVC, Prometh, jTracer),
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
