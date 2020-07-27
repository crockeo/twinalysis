package tweets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/dghubble/go-twitter/twitter"

	"github.com/crockeo/twinalysis/module"
)

const (
	DEFAULT_PERMS os.FileMode = 0755
)

func fetchTweets(
	client *twitter.Client,
	tweetEntryChan chan<- module.TweetEntry,
	username, cacheDir string,
	maxCachedID int64,
) error {
	errChan := make(chan error)
	defer close(errChan)

	quitChan := make(chan interface{})
	defer close(quitChan)

	tweetChan := make(chan twitter.Tweet)
	defer close(tweetChan)

	// collect tweets from the twitter API
	go func() {
		defer func () { quitChan <- 0 }()

		excludeReplies := false
		includeRetweets := false

		var maxID int64 = 1
		for {
			tweetBatch, _, err := client.Timelines.UserTimeline(
				&twitter.UserTimelineParams{
					ScreenName:      username,
					MaxID:           maxID - 1,
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
				return
			}

			var tweet twitter.Tweet
			for _, tweet = range tweetBatch {
				if tweet.ID <= maxCachedID {
					return
				}

				tweetChan <- tweet
				tweetEntryChan <- module.TweetEntry{
					Tweet:    tweet,
					Username: username,
				}
			}
			maxID = tweet.ID
		}
	}()

	// save tweets as they come in from tweetChan
	go func() {
		for tweet := range tweetChan {
			data, err := json.Marshal(tweet)
			if err != nil {
				return
			}

			err = ioutil.WriteFile(
				path.Join(cacheDir, fmt.Sprintf("%d.json", tweet.ID)),
				data,
				DEFAULT_PERMS,
			)
			if err != nil {
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

func CollectTweets(client *twitter.Client, tweetEntryChan chan<- module.TweetEntry, usernames []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(usernames))

	errChan := make(chan error)
	defer close(errChan)

	for _, username := range usernames {
		go func(username string) {
			defer wg.Done()

			cacheDir := path.Join(cwd, "data", username)
			files, err := ioutil.ReadDir(cacheDir)
			if err != nil {
				err = os.MkdirAll(cacheDir, DEFAULT_PERMS)
				if err != nil {
					errChan <- err
					return
				}
				files = []os.FileInfo{}
			}

			// reads contents of existing cache
			var maxCachedID int64
			for _, file := range files {
				contents, err := ioutil.ReadFile(path.Join(cacheDir, file.Name()))
				if err != nil {
					errChan <- err
					return
				}

				var tweet twitter.Tweet
				err = json.Unmarshal(contents, &tweet)
				if err != nil {
					errChan <- err
					return
				}

				tweetEntryChan <- module.TweetEntry{
					Tweet:    tweet,
					Username: username,
				}

				if maxCachedID < tweet.ID {
					maxCachedID = tweet.ID
				}
			}

			// fetches new tweets after the provided ID
			err = fetchTweets(client, tweetEntryChan, username, cacheDir, maxCachedID)
			if err != nil {
				errChan <- err
			}
		}(username)
	}

	wg.Wait()
	select {
	case err = <- errChan:
		return err
	default:
	}

	return nil
}
