package server

import (
	"net/http"
	"time"
)

const (
	defaultWriteTimeout = time.Second * 60
	defaultReadTimeout  = time.Second * 60
	defaultIdleTimeout  = time.Second * 20
)

func NewServer(serverAddress string, router http.Handler) *http.Server {
	serv := &http.Server{
		Addr:         serverAddress,
		Handler:      router,
		WriteTimeout: defaultWriteTimeout,
		ReadTimeout:  defaultReadTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	return serv
}
