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

func FormatNewPollReply(poll *votingpb.Poll) string {
	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf(":ballot_box: **A new anime poll has started** `%s`\n", poll.ShortId))

	if poll.Title != "" {
		buffer.WriteString(fmt.Sprintf("\n**%s**\n", poll.Title))
	}

	if poll.Expires > (time.Time{}).Unix() {
		formatedTime := time.Unix(poll.Expires, 0).UTC().Format("Jan 2, 2006 15:04:05 MST")

		buffer.WriteString(fmt.Sprintf("\nPoll ends `%s`\n", formatedTime))
	}

	buffer.WriteString(fmt.Sprintf("\nView the poll with `!poll` or `!poll -id %s`", poll.ShortId))

	return buffer.String()
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
