package gateway

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// A second concurrent burst should reuse the first burst's upstream sockets.
// The default two idle connections causes almost an entire burst to reconnect.
func TestProxyReusesConnectionsAcrossConcurrentBursts(t *testing.T) {
	const concurrency = 20
	arrived := make(chan struct{}, concurrency)
	release := make(chan struct{})
	var connections atomic.Int32
	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		arrived <- struct{}{}
		<-release
		io.WriteString(w, "ok")
	}))
	upstream.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	upstream.Start()
	defer upstream.Close()
	proxy, err := makeProxy(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Transport.(*http.Transport).CloseIdleConnections()
	for round := 0; round < 2; round++ {
		var wg sync.WaitGroup
		for n := 0; n < concurrency; n++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				out := httptest.NewRecorder()
				proxy.ServeHTTP(out, httptest.NewRequest("GET", "/api/me", nil))
				if out.Code != 200 || out.Body.String() != "ok" {
					t.Errorf("request failed: %d %s", out.Code, out.Body)
				}
			}()
		}
		for n := 0; n < concurrency; n++ {
			select {
			case <-arrived:
			case <-time.After(5 * time.Second):
				close(release)
				wg.Wait()
				t.Fatal("burst did not reach upstream")
			}
		}
		close(release)
		wg.Wait()
		if round == 0 {
			release = make(chan struct{})
		}
	}
	if got := connections.Load(); got != concurrency {
		t.Fatalf("created %d sockets for two %d-request bursts; expected connection reuse", got, concurrency)
	}
}
