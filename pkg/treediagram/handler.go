package treediagram

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jukeizu/contract"
	"github.com/jukeizu/voting/api/protobuf-spec/votingpb"
	"github.com/rs/zerolog"
	"github.com/shawntoffel/anilist"
)

type Handler struct {
	logger        zerolog.Logger
	votingClient  votingpb.VotingClient
	anilistClient anilist.Anilist
	httpServer    *http.Server
}

func NewHandler(logger zerolog.Logger, votingClient votingpb.VotingClient, anilistClient anilist.Anilist, addr string) Handler {
	logger = logger.With().Str("component", "intent.endpoint.anipoll").Logger()

	httpServer := http.Server{
		Addr: addr,
	}

	return Handler{logger, votingClient, anilistClient, &httpServer}
}

func (h Handler) CreateAnipoll(request contract.Request) (*contract.Response, error) {
	createAniPollRequest, err := ParseCreateAnipollRequest(request)
	if err != nil {
		return FormatParseError(err)
	}

	createPollRequest := createAniPollRequest.CreatePollRequest

	options, err := h.animeOptions(createAniPollRequest.Season, createAniPollRequest.Year)
	if err != nil {
		return FormatClientError(err)
	}

	for _, option := range options {
		createPollRequest.Options = append(createPollRequest.Options, option)
	}

	reply, err := h.votingClient.CreatePoll(context.Background(), createPollRequest)
	if err != nil {
		return FormatClientError(err)
	}

	return contract.StringResponse(FormatNewPollReply(reply.Poll)), nil
}

func (h Handler) animeOptions(season string, year string) ([]*votingpb.Option, error) {
	variables := map[string]string{}

	variables["season"] = strings.ToUpper(season)
	variables["seasonYear"] = year

	request := anilist.Request{
		Query:     anilist.DefaultAnimeForSeasonQuery,
		Variables: variables,
	}

	options := []*votingpb.Option{}

	for {
		response, err := h.anilistClient.Query(request)
		if err != nil {
			return options, err
		}

		if len(response.Errors) > 0 {
			return options, fmt.Errorf("anilist error: %v", response.Errors)
		}

		for _, anime := range response.Data.Page.Media {
			option := &votingpb.Option{
				Content: anime.Title.Romaji,
				Url:     anime.SiteUrl,
			}

			options = append(options, option)
		}

		if !response.Data.Page.PageInfo.HasNextPage {
			break
		}

		request.Variables["page"] = fmt.Sprintf("%d", response.Data.Page.PageInfo.CurrentPage+1)
	}

	return options, nil
}

func (h Handler) Start() error {
	h.logger.Info().Msg("starting")

	mux := http.NewServeMux()
	mux.HandleFunc("/anipoll", h.makeLoggingHttpHandlerFunc("anipoll", h.CreateAnipoll))

	h.httpServer.Handler = mux

	return h.httpServer.ListenAndServe()
}

func (h Handler) Stop() error {
	h.logger.Info().Msg("stopping")

	return h.httpServer.Shutdown(context.Background())
}

func (h Handler) makeLoggingHttpHandlerFunc(name string, f func(contract.Request) (*contract.Response, error)) http.HandlerFunc {
	contractHandlerFunc := contract.MakeRequestHttpHandlerFunc(f)

	return func(w http.ResponseWriter, r *http.Request) {
		defer func(begin time.Time) {
			h.logger.Info().
				Str("intent", name).
				Str("took", time.Since(begin).String()).
				Msg("called")
		}(time.Now())

		contractHandlerFunc.ServeHTTP(w, r)
	}
}
