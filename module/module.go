package module

import "github.com/dghubble/go-twitter/twitter"

type TweetEntry struct {
	Tweet    twitter.Tweet
	Username string
}

type Module interface {
	// Name returns the friendly name of this module. This name is used to execute this module from
	// the command line.
	Name() string

	// AnalyzeTweets performs some analysis over a collection of tweets.
	AnalyzeTweets(<-chan TweetEntry, chan<- error)
}
