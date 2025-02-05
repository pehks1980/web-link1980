package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/app/service"

	_ "github.com/pehks1980/go_gb_be1_kurs/web-link/internal/app/config"
	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/app/endpoint"
	"github.com/pehks1980/go_gb_be1_kurs/web-link/internal/pkg/repository"

	_ "go.uber.org/zap"

	// репозиторий (хранилище)  файло json or pg sql(db)

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/otel/semconv/v1.22.0"

	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	tracer "go.opentelemetry.io/otel/trace"
)

func initTracer() (*trace.TracerProvider, error) {
	// Initialize OTLP HTTP Exporter (Sends traces to Jaeger via OTLP)
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint("192.168.1.204:4318"),
		otlptracehttp.WithInsecure(), // Remove TLS for local testing
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
	}

	// Define the Tracer Provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter), // Send traces in batches
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("weblink"), // Change to your service name
		)),
	)

	// Set as the global tracer provider
	otel.SetTracerProvider(tp)

	return tp, nil
}

// главная петля
func main() {
	log.Print("Starting the app...")
	// настройка порта, настроек хранилища, таймаут при закрытии сервиса
	portdef := flag.String("port", "8000", "Port. Also, it can be set as env PORT.")

	storageType := flag.String("storage type", "pg", "data storage type: 'file' or 'pg'")

	storageNameDef := flag.String("storage name", "postgres://postuser:postpassword@192.168.1.204:5432/a4",
		"pg: 'postgres://dbuser:dbpasswd@ip_address:port/dbname'  file: 'storage.json'. It can be also set as REPO.")

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

	// Initialize OpenTelemetry Tracer
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("failed to shutdown tracer: %v", err)
		}
	}()

	// Create a new tracer
	jTracer := otel.Tracer("weblink")

	port := os.Getenv("PORT")
	if port == "" {
		log.Printf("$PORT is not set. using default %s", *portdef)
		port = *portdef
	} else {
		log.Printf("Using env $PORT = %s", port)
	}

	storageName := os.Getenv("REPO")
	if storageName == "" {
		log.Printf("$REPO is not set. using default %s", *storageNameDef)
		storageName = *storageNameDef
	} else {
		log.Printf("Using env $REPO = %s", storageName)
	}

	// инициализация файлового хранилища ук на структуру repo
	var repoif, linkSVC repository.RepoIf

	// create empty context for this app
	ctx := context.Background()
	// подстановка в интерфейс соотвествующего хранилища
	if *storageType == "file" {
		repoif = new(repository.FileRepo)
	}
	if *storageType == "pg" {
		repoif = new(repository.PgRepo)
	}
	// init selected repo interface (file or pg)
	repoif = repoif.New(ctx, storageName, jTracer)
	defer repoif.CloseConn()
	// init cache service interface which works as shim between selected repo and http handlers
	// service interface provides redis cache feature
	//linkSVC = service.New(repoif, jTracer) //cache aside
	linkSVC = service.NewWb(repoif, jTracer) //cache aside + cache write back with async workers
	// такая схема получается
	// DB(file) repoif <-> cache service (service/servicewb) linkSVC <-> API (endpoint) <-> http:8080

	// Prometheus init //////////////////////////////////
	// создаем структуру-интерфейс для прометиуса, включающую 2 обьекта cчетчик и гистограммка
	var promif, Prometh endpoint.PromIf

	promif = new(endpoint.Prom)
	Prometh = promif.New()

	//init our appsvc struct
	appsvc := endpoint.NewAppsvc(linkSVC, Prometh, jTracer)

	serv := http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: endpoint.RegisterPublicHTTP(appsvc),
	}

	// tracer first message
	_, span := jTracer.Start(ctx, "jTracer.Start:")
	defer span.End()

	timeoutStr := strconv.FormatInt(*shutdownTimeout, 10)

	span.AddEvent("main inits: ", tracer.WithAttributes(
		attribute.String("API port", *portdef),
		attribute.String("repo", *storageType),
		attribute.String("repo def", *storageNameDef),
		attribute.String("shutdown timeout", timeoutStr),
	))

	defer func() {
		ctxf, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := tp.ForceFlush(ctxf); err != nil {
			log.Fatalf("failed to flush traces: %v", err)
		}
	}()

	// запуск сервера
	go func() {
		// ignore standard error when gracefull shutdown - Server closed
		if err := serv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve err: %v", err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	log.Printf("Started app at port = %s", port)
	// ждет сигнала
	sig := <-interrupt

	log.Printf("Sig: %v, stopping app", sig)

	linkSVC.CloseConn()
	// шат даун по контексту с тайм аутом
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*shutdownTimeout)*time.Second)
	defer cancel()
	if err := serv.Shutdown(ctx); err != nil {
		log.Printf("shutdown err: %v", err)
	}

}
