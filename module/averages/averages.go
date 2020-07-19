package averages

import (
	"os"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/jedib0t/go-pretty/v6/table"
)

type Averages struct {
}

func (a Averages) Name() string {
	return "averages"
}

func (a Averages) AnalyzeTweets(tweetsByUsername map[string][]twitter.Tweet) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(
		table.Row{"Username", "Tweets", "Favorites", "Retweets", "Replies", "Quotes"},
	)
	for username, tweets := range tweetsByUsername {
		tweetCount := 0
		favorites := 0
		retweets := 0
		replies := 0
		quotes := 0
		for _, tweet := range tweets {
			tweetCount++
			favorites += tweet.FavoriteCount
			retweets += tweet.RetweetCount
			replies += tweet.ReplyCount
			quotes += tweet.QuoteCount
		}

		norm := float32(len(tweets))
		t.AppendRow(
			table.Row{
				username,
				tweetCount,
				float32(favorites) / norm,
				float32(retweets) / norm,
				float32(replies) / norm,
				float32(quotes) / norm,
			},
		)
	}
	t.Render()

	return nil
}
