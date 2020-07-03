package treediagram

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jukeizu/contract"
	"github.com/jukeizu/voting/api/protobuf-spec/votingpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var CountdownURL = "https://countdown.treediagram.xyz"
var ThumbnailURL = "https://cdn.discordapp.com/attachments/320660733740449792/728375524090576996/ff85a1aae50ad48506e3275656768e89.png"

func FormatNewPollReply(poll *votingpb.Poll) *contract.Embed {
	embed := &contract.Embed{
		Color:        0x5dadec,
		Title:        fmt.Sprintf("**A new anime poll has started** `%s`\n", poll.ShortId),
		ThumbnailUrl: ThumbnailURL,
	}

	buffer := bytes.Buffer{}

	if poll.Title != "" {
		buffer.WriteString(fmt.Sprintf("\n**%s**\n", poll.Title))
	}

	if poll.Expires > (time.Time{}).Unix() {
		t := time.Unix(poll.Expires, 0).UTC()
		displayTime := t.Format("Jan 2, 2006 15:04 MST")

		buffer.WriteString(fmt.Sprintf("\nPoll ends: [%s](%s?t=%d)\n", displayTime, CountdownURL, poll.Expires))
	}

	buffer.WriteString(fmt.Sprintf("\nView the poll with `!poll` or `!poll -id %s`", poll.ShortId))

	embed.Description = buffer.String()

	return embed
}

func FormatParseError(err error) (*contract.Response, error) {
	switch err.(type) {
	case ParseError:
		return contract.StringResponse(err.Error()), nil
	}

	return nil, err
}

func FormatClientError(err error) (*contract.Response, error) {
	st, ok := status.FromError(err)
	if !ok {
		return nil, err
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return contract.StringResponse(st.Message()), nil
	case codes.NotFound:
		return contract.StringResponse(st.Message()), nil
	}

	return nil, err
}
