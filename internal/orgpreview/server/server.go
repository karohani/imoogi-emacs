package server

import (
	"context"
	"net"
	"net/http"

	"github.com/karohani/imoogi-emacs/internal/orgpreview"
)

type Config struct {
	Addr         string
	Token        string
	AllowedRoots []string
}

type Instance struct {
	URL string
	srv *http.Server
}

func Start(ctx context.Context, cfg Config) (*Instance, error) {
	core, err := orgpreview.NewServer(orgpreview.ServerConfig{
		Token:        cfg.Token,
		AllowedRoots: cfg.AllowedRoots,
		Parser:       orgpreview.FallbackParser{},
	})
	if err != nil {
		return nil, err
	}
	httpServer, ln, err := core.Listen(ctx, cfg.Addr)
	if err != nil {
		return nil, err
	}
	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		_ = httpServer.Shutdown(context.Background())
		return nil, err
	}
	if host == "" || host == "::1" || host == "localhost" {
		host = "127.0.0.1"
	}
	return &Instance{URL: "http://" + net.JoinHostPort(host, port), srv: httpServer}, nil
}

func (i *Instance) Close() error {
	return i.srv.Shutdown(context.Background())
}
