package tweets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/dghubble/go-twitter/twitter"
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
func CollectTweets(client *twitter.Client, username string) ([]twitter.Tweet, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return []twitter.Tweet{}, nil
	}
	tweetDir := path.Join(cwd, "data", username)

	if _, err := os.Stat(tweetDir); os.IsNotExist(err) {
		err = os.MkdirAll(tweetDir, DEFAULT_PERMS)
		if err != nil {
			return []twitter.Tweet{}, err
		}

		err = fetchTweets(
			client,
			username,
			tweetDir,
		)
		if err != nil {
			return []twitter.Tweet{}, err
		}
	}

	files, err := ioutil.ReadDir(tweetDir)
	if err != nil {
		return []twitter.Tweet{}, nil
	}

	tweets := make([]twitter.Tweet, len(files))
	for i, file := range files {
		tweet, err := readTweet(
			path.Join(tweetDir, file.Name()),
		)
		if err != nil {
			return []twitter.Tweet{}, err
		}
		tweets[i] = tweet
	}

	return tweets, nil
}
