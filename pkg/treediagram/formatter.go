package treediagram

import (
	"fmt"
	"time"

	"github.com/jukeizu/contract"
	"github.com/jukeizu/voting/api/protobuf-spec/votingpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var CountdownURL = "https://countdown.treediagram.xyz"
var BallotBoxThumbnailURL = "https://cdn.discordapp.com/attachments/320660733740449792/728375524090576996/ff85a1aae50ad48506e3275656768e89.png"

func FormatNewPollReply(poll *votingpb.Poll) *contract.Message {
	embed := &contract.Embed{
		Color:        0x5865f2,
		Title:        "**A new anime poll has started**",
		ThumbnailUrl: BallotBoxThumbnailURL,
		Footer: &contract.EmbedFooter{
			Text: poll.ShortId,
		},
	}

	if poll.Title != "" {
		embed.Fields = append(embed.Fields, &contract.EmbedField{
			Name:  "Title",
			Value: poll.Title,
		})
	}

	if hasExpiration(poll) {
		embed.Fields = append(embed.Fields, &contract.EmbedField{
			Name:  "Open until",
			Value: fmt.Sprintf("<t:%d>", poll.Expires),
		})
	}

	embed.Fields = append(embed.Fields, &contract.EmbedField{
		Name:  "Started by",
		Value: fmt.Sprintf("<@!%s>", poll.CreatorId),
	})

	return &contract.Message{
		Embed: embed,
		Compontents: &contract.Components{
			ActionsRows: []*contract.ActionsRow{
				&contract.ActionsRow{
					Buttons: []*contract.Button{
						&contract.Button{
							Label:    "Vote",
							CustomId: fmt.Sprintf("poll.%s", poll.ShortId),
							Emoji: contract.ComponentEmoji{
								Name: "🗳️",
							},
						},
					},
				},
			},
		},
	}
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

func hasExpiration(poll *votingpb.Poll) bool {
	return poll.Expires > (time.Time{}).Unix()
}
