package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"github.com/karohani/imoogi-emacs/internal/orgpreview"
)

func main() {
	addr := flag.String("addr", "", "loopback listen address")
	host := flag.String("host", "127.0.0.1", "loopback listen host")
	port := flag.Int("port", 0, "loopback listen port")
	token := flag.String("token", "", "session token; generated when empty")
	printBootstrapJSON := flag.Bool("print-bootstrap-json", false, "print one bootstrap JSON line for the Emacs client")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	server, err := orgpreview.NewServer(orgpreview.ServerConfig{Token: *token, Parser: orgpreview.FallbackParser{}})
	if err != nil {
		log.Fatal(err)
	}
	listenAddr := *addr
	if listenAddr == "" {
		listenAddr = net.JoinHostPort(*host, strconv.Itoa(*port))
	}
	httpServer, ln, err := server.Listen(ctx, listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	if *printBootstrapJSON {
		host, port, err := net.SplitHostPort(ln.Addr().String())
		if err != nil {
			log.Fatal(err)
		}
		portNumber, err := strconv.Atoi(port)
		if err != nil {
			log.Fatal(err)
		}
		bootstrap := map[string]any{
			"addr":  ln.Addr().String(),
			"host":  host,
			"port":  portNumber,
			"token": server.Token(),
		}
		if err := json.NewEncoder(os.Stdout).Encode(bootstrap); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("addr=%s token=%s\n", ln.Addr().String(), server.Token())
	}
	<-ctx.Done()
	if err := httpServer.Shutdown(context.Background()); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
