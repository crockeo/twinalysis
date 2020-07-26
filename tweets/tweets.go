package tweets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/dghubble/go-twitter/twitter"

	"github.com/crockeo/twinalysis/module"
)

const (
	DEFAULT_PERMS os.FileMode = 0755
)

// fetchTweets fetches the tweets of a particular user and saves them to disk in the directory
// provided.
func fetchTweets(client *twitter.Client, username string, tweetDir string) error {
	errChan := make(chan error)
	defer close(errChan)

	quitChan := make(chan interface{})
	defer close(quitChan)

	tweetChan := make(chan twitter.Tweet)
	defer close(tweetChan)

	// collect tweets from the twitter API
	go func() {
		excludeReplies := false
		includeRetweets := false

		var maxID int64 = 1
		for {
			tweetBatch, _, err := client.Timelines.UserTimeline(
				&twitter.UserTimelineParams{
					ScreenName:      username,
					MaxID:           maxID - 1, // Only show new tweets
					ExcludeReplies:  &excludeReplies,
					IncludeRetweets: &includeRetweets,
					TweetMode:       "extended",
				},
			)
			if err != nil {
				errChan <- err
				return
			}
			if len(tweetBatch) == 0 {
				break
			}

			var tweet twitter.Tweet
			for _, tweet = range tweetBatch {
				tweetChan <- tweet
			}
			maxID = tweet.ID
		}

		quitChan <- 0
	}()

	// save tweets as they come in from tweetChan
	go func() {
		for tweet := range tweetChan {
			data, err := json.Marshal(tweet)
			if err != nil {
				errChan <- err
				return
			}

			err = ioutil.WriteFile(
				path.Join(tweetDir, fmt.Sprintf("%d.json", tweet.ID)),
				data,
				DEFAULT_PERMS,
			)
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-quitChan:
		return nil
	}
}

// readTweet reads a single tweet from disk.
func readTweet(path string) (twitter.Tweet, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return twitter.Tweet{}, err
	}

	tweet := twitter.Tweet{}
	err = json.Unmarshal(data, &tweet)
	if err != nil {
		return twitter.Tweet{}, err
	}

	return tweet, nil
}

// CollectTweets collects all tweets from a particular user. This may be sourced from querying the
// Twitter API or from a local on-disk cache.
func CollectTweets(client *twitter.Client, tweetEntryChan chan<- module.TweetEntry, usernames []string) error {
	// TODO: Optimize this whole thing.
	//
	// Current architecture:
	//   1. If cache does not exist
	//     1.a. Spin up gofunc to retrieve tweets
	//     1.b. Spin up gofunc to save tweets
	//     1.c. Wait until all tweets have been saved to disk
	//   2. Read tweets from disk
	//
	// Problems:
	//   1. Never updates the cache (never checks new tweets / never updates data for old tweets)
	//   2. Saves tweet to disk for no reason, could just be directly funneled out the tweetEntryChan
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	for _, username := range usernames {
		tweetDir := path.Join(cwd, "data", username)

		if _, err := os.Stat(tweetDir); os.IsNotExist(err) {
			err = os.MkdirAll(tweetDir, DEFAULT_PERMS)
			if err != nil {
				return err
			}

			if err = fetchTweets(client, username, tweetDir); err != nil {
				return err
			}
		}

		files, err := ioutil.ReadDir(tweetDir)
		if err != nil {
			return nil
		}

		for _, file := range files {
			tweet, err := readTweet(path.Join(tweetDir, file.Name()))
			if err != nil {
				return err
			}
			tweetEntryChan <- module.TweetEntry{
				Username: username,
				Tweet: tweet,
			}
		}
	}

	close(tweetEntryChan)
	return nil
}
