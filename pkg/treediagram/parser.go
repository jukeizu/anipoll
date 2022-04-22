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

var validSeasons = []string{
	"SPRING",
	"SUMMER",
	"WINTER",
	"FALL",
}

var validFormats = []string{
	"TV",
	"TV_SHORT",
	"MOVIE",
	"SPECIAL",
	"OVA",
	"ONA",
	"MUSIC",
	"MANGA",
	"NOVEL",
	"ONE_SHOT",
}

type AnipollRequest struct {
	CreatePollRequest *votingpb.CreatePollRequest
	Season            string
	Year              string
	Formats           []string
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
	defaultFormats := "TV,ONA"

	title := parser.String("t", "", "The poll title")
	allowedUniqueVotes := parser.Int("n", 0, "The number of unique votes a user can submit. (defaults to the number of anime + additional options)")
	parsedSeason := parser.String("s", defaultSeason, "The anime season")
	year := parser.String("y", defaultYear, "The anime year")
	parsedFormats := parser.String("f", defaultFormats, "comma delimited list of anime formats")
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

	season := strings.ToUpper(*parsedSeason)
	if len(season) > 0 && !isValid(season, validSeasons) {
		return AnipollRequest{}, ParseError{Message: "Invalid season provided. Must be one of: `" + strings.Join(validSeasons, ", ") + "`"}
	}

	t, err := parseEndTime("1/2/06 15:04", *ends)
	if err != nil {
		return AnipollRequest{}, ParseError{Message: err.Error()}
	}

	if *ends != "" && t.Before(time.Now().UTC()) {
		return AnipollRequest{}, ParseError{Message: "Poll end time must be in the future."}
	}

	formats := strings.Split(*parsedFormats, ",")
	for i := range formats {
		format := strings.ToUpper(strings.TrimSpace(formats[i]))
		if !isValid(format, validFormats) {
			return AnipollRequest{}, ParseError{Message: "Invalid format provided. Must be one of: `" + strings.Join(validFormats, ", ") + "`"}
		}
		formats[i] = format
	}

	createPollRequest := &votingpb.CreatePollRequest{
		Title:              buildTitle(*title, season, *year),
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
		Season:            season,
		Year:              *year,
		Formats:           formats,
	}

	return anipollRequest, nil
}

func buildTitle(title string, season string, year string) string {
	if title != "" {
		return title
	}
	return fmt.Sprintf("%s %s", strings.Title(strings.ToLower(season)), year)
}

func isValid(intput string, valid []string) bool {
	for _, item := range valid {
		if intput == item {
			return true
		}
	}

	return false
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
