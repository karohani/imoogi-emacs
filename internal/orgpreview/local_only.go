package orgpreview

import (
	"net"
	"net/http"
)

// localOnly admits requests addressed to this listener's loopback port and, when
// a browser names an origin, only this page's own origin. It shuts out DNS
// rebinding (a foreign Host) and other sites' pages (a foreign Origin), so a
// leaked preview URL is not usable from an arbitrary web page.
func localOnly(port string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, hostPort, err := net.SplitHostPort(r.Host)
		if err != nil || hostPort != port || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
