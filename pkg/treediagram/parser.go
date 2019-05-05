package treediagram

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/jukeizu/contract"
	"github.com/jukeizu/voting/api/protobuf-spec/votingpb"
	shellwords "github.com/mattn/go-shellwords"
)

type AnipollRequest struct {
	CreatePollRequest *votingpb.CreatePollRequest
	Season            string
	Year              string
}

func ParseCreateAnipollRequest(request contract.Request) (AnipollRequest, error) {
	args, err := shellwords.Parse(request.Content)
	if err != nil {
		return AnipollRequest{}, err
	}

	outputBuffer := bytes.NewBuffer([]byte{})

	parser := flag.NewFlagSet("anipoll", flag.ContinueOnError)
	parser.SetOutput(outputBuffer)

	defaultYear := fmt.Sprintf("%d", time.Now().Year())

	title := parser.String("t", "", "The poll title")
	allowedUniqueVotes := parser.Int("n", 0, "The number of unique votes a user can submit. Default is max.")
	season := parser.String("s", "", "The anime season. (winter, spring, summer, fall)")
	year := parser.String("y", defaultYear, "The anime year.")

	err = parser.Parse(args[1:])
	if err != nil {
		return AnipollRequest{}, ParseError{Message: outputBuffer.String()}
	}

	if len(*season) > 0 && !isValidSeason(*season) {
		return AnipollRequest{}, ParseError{Message: "That season is not valid."}
	}

	createPollRequest := &votingpb.CreatePollRequest{
		Title:              buildTitle(*title, *season, *year),
		AllowedUniqueVotes: int32(*allowedUniqueVotes),
		ServerId:           request.ServerId,
		CreatorId:          request.Author.Id,
	}

	for _, content := range parser.Args() {
		option := &votingpb.Option{
			Content: content,
		}

		createPollRequest.Options = append(createPollRequest.Options, option)
	}

	anipollRequest := AnipollRequest{
		CreatePollRequest: createPollRequest,
		Season:            *season,
		Year:              *year,
	}

	return anipollRequest, nil
}

func buildTitle(title string, season string, year string) string {
	if title != "" {
		return title
	}
	return fmt.Sprintf("%s %s", strings.Title(season), year)
}

func isValidSeason(season string) bool {
	seasons := map[string]bool{}
	seasons["spring"] = true
	seasons["summer"] = true
	seasons["winter"] = true
	seasons["fall"] = true

	_, validSeason := seasons[season]

	return validSeason
}
