package treediagram

import (
	"context"
	"fmt"
	"net/http"
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

	animeOptions, err := h.animeOptions(createAniPollRequest.Season, createAniPollRequest.Year, createAniPollRequest.Formats)
	if err != nil {
		h.logger.Error().Err(err).Caller().Msg("received an error from anilist")

		return contract.StringResponse("the AniList API is unavailable at the moment :anguished:"), nil
	}

	for _, option := range animeOptions {
		createPollRequest.Options = append(createPollRequest.Options, option)
	}

	if createPollRequest.AllowedUniqueVotes == 0 {
		createPollRequest.AllowedUniqueVotes = int32(len(createPollRequest.Options))
	}

	reply, err := h.votingClient.CreatePoll(context.Background(), createPollRequest)
	if err != nil {
		h.logger.Error().Err(err).Caller().Msg("failed to create poll")
		return FormatClientError(err)
	}

	message := FormatNewPollReply(reply.Poll)

	return &contract.Response{Messages: []*contract.Message{message}}, nil
}

func (h Handler) animeOptions(season string, year string, formats []string) ([]*votingpb.Option, error) {
	request := anilist.Request{
		Query: anilist.DefaultAnimeForSeasonQuery,
		Variables: map[string]interface{}{
			"season":     season,
			"seasonYear": year,
			"formats":    formats,
		},
	}

	options := []*votingpb.Option{}

	for {
		response, err := h.anilistClient.Query(context.Background(), request)
		if err != nil {
			return options, err
		}

		if len(response.Errors) > 0 {
			return options, fmt.Errorf("anilist error: %v", response.Errors)
		}

		for _, anime := range response.Data.Page.Media {
			if anime.Episodes != 0 && anime.Episodes < 5 {
				continue
			}
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
