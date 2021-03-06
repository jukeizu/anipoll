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

	endTime := time.Now().UTC().Add(time.Hour * 120).Format("1/2/06 15:04")

	defaultSeason := defaultSeason()
	defaultYear := fmt.Sprintf("%d", time.Now().Year())

	title := parser.String("t", "", "The poll title")
	allowedUniqueVotes := parser.Int("n", 0, "The number of unique votes a user can submit. (defaults to the number of anime + additional options)")
	season := parser.String("s", defaultSeason, "The anime season")
	year := parser.String("y", defaultYear, "The anime year")
	ends := parser.String("ends", endTime, fmt.Sprintf("The UTC end time for the poll. (format \"M/d/yy H:mm\")"))

	parser.Usage = func() {
		fmt.Fprintf(parser.Output(), "Usage of %s:\n", parser.Name())
		parser.PrintDefaults()
		fmt.Fprintln(parser.Output(), "[options]... \n    \tAdditional poll options. Must come after all other options.")
	}

	err = parser.Parse(args[1:])
	if err != nil {
		return AnipollRequest{}, ParseError{Message: outputBuffer.String()}
	}

	if len(*season) > 0 && !isValidSeason(*season) {
		return AnipollRequest{}, ParseError{Message: "That season is not valid."}
	}

	t, err := parseEndTime("1/2/06 15:04", *ends)
	if err != nil {
		return AnipollRequest{}, ParseError{Message: err.Error()}
	}

	if *ends != "" && t.Before(time.Now().UTC()) {
		return AnipollRequest{}, ParseError{Message: "Poll end time must be in the future."}
	}

	createPollRequest := &votingpb.CreatePollRequest{
		Title:              buildTitle(*title, *season, *year),
		AllowedUniqueVotes: int32(*allowedUniqueVotes),
		ServerId:           request.ServerId,
		CreatorId:          request.Author.Id,
		Expires:            t.Unix(),
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
	seasons := map[string]bool{
		"spring": true,
		"summer": true,
		"winter": true,
		"fall":   true,
	}
	_, validSeason := seasons[strings.ToLower(season)]

	return validSeason
}

func defaultSeason() string {
	month := time.Now().Month()

	switch month {
	case time.January, time.February, time.March:
		return "winter"
	case time.April, time.May, time.June:
		return "spring"
	case time.July, time.August, time.September:
		return "summer"
	case time.October, time.November, time.December:
		return "fall"
	}

	return ""
}

func parseEndTime(layout string, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	return time.Parse(layout, value)
}
