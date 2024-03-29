package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/options"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/version"
)

var collectorArgs []string

func TestMain(m *testing.M) {
	// TODO should get these from env vars or flags
	//version = os.Getenv("VERSION")
	//commit = os.Getenv("GIT_COMMIT")

	version.Version = "1.12.0"
	version.Commit = "4930b29"

	fmt.Println(fmt.Sprintf("attempting to run test collector for coverage data with version '%s' and commit '%s'", version.Version, version.Commit))

	fmt.Println(fmt.Sprintf("arg stuff BEFORE shenanigans: collectorArgs '%+v' os.Args '%+v'", collectorArgs, os.Args))
	collectorArgs = os.Args[2:]
	os.Args = os.Args[:2]
	fmt.Println(fmt.Sprintf("arg stuff AFTER shenanigans: collectorArgs '%+v' os.Args '%+v'", collectorArgs, os.Args))

	os.Exit(m.Run())
}

func TestMainCoverage(t *testing.T) {
	// TODO consider making this more legit
	if collectorArgs[0] != "--daemon" {
		t.Skip("skipping collector coverage test: it appears a normal go test is being run")
	}

	ctx, cancel := context.WithCancel(context.Background())
	ks := newKillServer(":19999", cancel)
	go ks.Start()

	os.Args = append([]string{"./wavefront-collector.test"}, collectorArgs...)
	go main()

	util.SetAgentType(options.AllAgentType)

	<-ctx.Done()

	fmt.Println("context done; attempting to shut down")
	ks.server.Shutdown(context.Background())
}

type killServer struct {
	server http.Server
	cancel context.CancelFunc
}

func newKillServer(addr string, cancel context.CancelFunc) *killServer {
	return &killServer{
		server: http.Server{
			Addr: addr,
		},
		cancel: cancel,
	}
}

func (s *killServer) Start() {
	s.server.Handler = s

	err := s.server.ListenAndServe()
	if err != nil {
		fmt.Println("KillServer Error:", err)
	}
}

func (s *killServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	fmt.Println("receiving kill curl; attempting context cancel")
	// cancel the context
	s.cancel()
}
