package averages

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/crockeo/twinalysis/module"
)

type tweetStats struct {
	tweetCount int
	favorites  int
	retweets   int
	replies    int
	quotes     int
}

func (ts *tweetStats) addTweetEntry(tweetEntry module.TweetEntry) {
	ts.tweetCount += 1
	ts.favorites += tweetEntry.Tweet.FavoriteCount
	ts.retweets += tweetEntry.Tweet.RetweetCount
	ts.replies += tweetEntry.Tweet.ReplyCount
	ts.quotes += tweetEntry.Tweet.QuoteCount
}

type Averages struct {
}

func (a Averages) Name() string {
	return "averages"
}

func (a Averages) AnalyzeTweets(tweetEntryChan <-chan module.TweetEntry) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(
		table.Row{"Username", "Tweets", "Favorites", "Retweets", "Replies", "Quotes"},
	)

	tweetStatsByUsername := map[string]*tweetStats{}
	for tweetEntry := range tweetEntryChan {
		if _, ok := tweetStatsByUsername[tweetEntry.Username]; !ok {
			tweetStatsByUsername[tweetEntry.Username] = &tweetStats{}
		}
		tweetStatsByUsername[tweetEntry.Username].addTweetEntry(tweetEntry)
	}

	for username, tweetStats := range tweetStatsByUsername {
		norm := float32(tweetStats.tweetCount)
		t.AppendRow(
			table.Row{
				username,
				tweetStats.tweetCount,
				float32(tweetStats.favorites) / norm,
				float32(tweetStats.retweets) / norm,
				float32(tweetStats.replies) / norm,
				float32(tweetStats.quotes) / norm,
			},
		)
	}
	t.Render()

	return nil
}
