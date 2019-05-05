package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpczerolog "github.com/cheapRoc/grpc-zerolog"
	_ "github.com/jnewmano/grpc-json-proxy/codec"
	"github.com/jukeizu/anipoll/pkg/treediagram"
	"github.com/jukeizu/voting/api/protobuf-spec/votingpb"
	"github.com/oklog/run"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/shawntoffel/anilist"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/keepalive"
)

var Version = ""

var (
	flagVersion = false
	flagHandler = true

	httpPort          = "10002"
	votingServiceAddr = "localhost:50053"
)

func parseConfig() {
	flag.StringVar(&httpPort, "http.port", httpPort, "http port for handler")
	flag.StringVar(&votingServiceAddr, "voting.addr", votingServiceAddr, "address of service if not local")
	flag.BoolVar(&flagHandler, "handler", flagHandler, "Run as handler")
	flag.BoolVar(&flagVersion, "v", false, "version")

	flag.Parse()
}

func main() {
	parseConfig()

	if flagVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().
		Str("instance", xid.New().String()).
		Str("component", "anipoll").
		Str("version", Version).
		Logger()

	grpcLoggerV2 := grpczerolog.New(logger.With().Str("transport", "grpc").Logger())
	grpclog.SetLoggerV2(grpcLoggerV2)

	clientConn, err := grpc.Dial(votingServiceAddr, grpc.WithInsecure(),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                30 * time.Second,
				Timeout:             10 * time.Second,
				PermitWithoutStream: true,
			},
		),
	)
	if err != nil {
		logger.Error().Err(err).Str("votingServiceAddr", votingServiceAddr).Msg("could not dial service address")
		os.Exit(1)
	}

	g := run.Group{}

	if flagHandler {
		votingClient := votingpb.NewVotingClient(clientConn)
		anilistClient := anilist.New()

		httpAddr := ":" + httpPort

		handler := treediagram.NewHandler(logger, votingClient, anilistClient, httpAddr)

		g.Add(func() error {
			return handler.Start()
		}, func(error) {
			err := handler.Stop()
			if err != nil {
				logger.Error().Err(err).Caller().Msg("couldn't stop handler")
			}
		})
	}

	cancel := make(chan struct{})
	g.Add(func() error {
		return interrupt(cancel)
	}, func(error) {
		close(cancel)
	})

	logger.Info().Err(g.Run()).Msg("stopped")
}

func interrupt(cancel <-chan struct{}) error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-cancel:
		return errors.New("stopping")
	case sig := <-c:
		return fmt.Errorf("%s", sig)
	}
}
