package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/httpapi"
	"seed-vigor-gate/internal/store"
)

type Runtime struct {
	Store    *store.Store
	HTTP     *http.Server
	Listener net.Listener
}

func BuildRuntime(config Config) (*Runtime, error) {
	if config.DatabasePath != ":memory:" {
		directory := filepath.Dir(config.DatabasePath)
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return nil, fmt.Errorf("创建数据目录: %w", err)
		}
	}
	repository, err := store.Open(config.DatabasePath)
	if err != nil {
		return nil, err
	}
	service := application.NewService(repository)
	api := httpapi.New(service)
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		repository.Close()
		return nil, fmt.Errorf("监听 %s: %w", config.Address, err)
	}
	httpServer := &http.Server{Addr: config.Address, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	return &Runtime{Store: repository, HTTP: httpServer, Listener: listener}, nil
}

func (r *Runtime) Serve(errors chan<- error) {
	err := r.HTTP.Serve(r.Listener)
	if err == http.ErrServerClosed {
		err = nil
	}
	errors <- err
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	httpErr := r.HTTP.Shutdown(ctx)
	storeErr := r.Store.Close()
	if httpErr != nil {
		return httpErr
	}
	return storeErr
}
