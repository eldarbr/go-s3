package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/eldarbr/go-auth/pkg/cache"
	"github.com/eldarbr/go-auth/pkg/config"
	"github.com/eldarbr/go-auth/pkg/database"
	"github.com/eldarbr/go-s3/internal/auth"
	"github.com/eldarbr/go-s3/internal/business"
	"github.com/eldarbr/go-s3/internal/handler"
	"github.com/eldarbr/go-s3/internal/provider/files"
	"github.com/eldarbr/go-s3/internal/server"
)

type appConfig struct {
	DBUri             string `yaml:"dbUri"`
	ServingURI        string `yaml:"servingUri"`
	PublicPemPath     string `yaml:"publicPemPath"`
	SslCertfilePath   string `yaml:"sslCertfilePath"`
	SslKeyfilePath    string `yaml:"sslKeyfilePath"`
	PprofServingURI   string `yaml:"pprofServingUri"`
	EnableTLSServing  bool   `yaml:"enableTlsServing"`
	StoragePath       string `yaml:"storagePath"`
	RateLimitRequests int    `yaml:"rateLimitRequests"`
	RateLimitTTL      int64  `yaml:"rateLimitTtl"`
	RateLimitCapacity int    `yaml:"rateLimitCapacity"`
}

const (
	CacheAutoEvictPeriodSeconds = 120
	filesStorageDirMode         = 0700
	filesStorageFileMode        = 0700
	ConfigPath                  = "secret/config.yaml"
	DBMigrationsPath            = "file://./sql"
)

// set defaults.
func newAppConfig() (conf appConfig) {
	return
}

func main() {
	programContext, programContextStop := signal.NotifyContext(context.Background(), syscall.SIGINT)

	defer programContextStop()

	conf := newAppConfig()

	err := config.ParseConfig(ConfigPath, &conf)
	if err != nil {
		log.Println(err)

		return
	}

	if conf.PprofServingURI != "" {
		log.Println("Starting pprof http")

		go func() {
			log.Println(http.ListenAndServe(conf.PprofServingURI, server.NewPprofServemux()))
		}()
	}

	jwtService, jwtErr := auth.NewJWTService(conf.PublicPemPath)
	if jwtErr != nil {
		log.Println(jwtErr)

		return
	}

	dbInstance, err := database.Setup(programContext, conf.DBUri, DBMigrationsPath)
	if err != nil {
		log.Println(err)

		return
	}

	log.Println("Database setup ok")

	cache := cache.NewCache(conf.RateLimitTTL, conf.RateLimitCapacity)

	go cache.AutoEvict(CacheAutoEvictPeriodSeconds * time.Second)

	var serv *http.Server
	{
		fileStorage := files.NewContainer(conf.StoragePath, filesStorageFileMode, filesStorageDirMode)
		business := business.NewBusinessModule(dbInstance, fileStorage)
		apiHandler := handler.NewAPIHandler(business, jwtService, cache, conf.RateLimitRequests)
		router := server.NewRouter(apiHandler)
		serv = server.NewServer(conf.ServingURI, router)
	}

	if conf.EnableTLSServing {
		go func() {
			err = serv.ListenAndServeTLS(conf.SslCertfilePath, conf.SslKeyfilePath)
		}()
	} else {
		go func() {
			err = serv.ListenAndServe()
		}()
	}

	<-programContext.Done()

	log.Println("shutting down")

	{
		shutdownContext, shutdownContextCancel := context.WithTimeout(programContext, time.Second*15)
		defer shutdownContextCancel()

		err = serv.Shutdown(shutdownContext)

		cache.StopAutoEvict()
		dbInstance.ClosePool()
	}

	if err != nil {
		log.Println(err)
	}
}
