package server

import (
	"net/http"
	"time"
)

const (
	defaultWriteTimeout = time.Second * 240
	defaultReadTimeout  = time.Second * 240
	defaultIdleTimeout  = time.Second * 240
)

func NewServer(serverAddress string, router http.Handler) *http.Server {
	serv := &http.Server{
		Addr:                         serverAddress,
		Handler:                      router,
		WriteTimeout:                 defaultWriteTimeout,
		ReadTimeout:                  defaultReadTimeout,
		IdleTimeout:                  defaultIdleTimeout,
		DisableGeneralOptionsHandler: true,
	}

	return serv
}
