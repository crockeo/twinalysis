package tweets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/dghubble/go-twitter/twitter"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/crockeo/twinalysis/module"
)

const (
	API_KEY_FILE        string = "secrets/api"
	API_SECRET_KEY_FILE string = "secrets/api_secret"
	BEARER_TOKEN_FILE   string = "secrets/bearer"

	DEFAULT_PERMS os.FileMode = 0755
)

func readKey(path string) (string, error) {
	rawKey, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(rawKey), nil
}

type Client struct {
	twitterClient  *twitter.Client
	tweetEntryChan chan module.TweetEntry
}

func NewClient() (*Client, error) {
	apiKey, err := readKey(API_KEY_FILE)
	if err != nil {
		return nil, err
	}
	apiKeySecret, err := readKey(API_SECRET_KEY_FILE)
	if err != nil {
		return nil, err
	}

	config := &clientcredentials.Config{
		ClientID:     apiKey,
		ClientSecret: apiKeySecret,
		TokenURL:     "https://api.twitter.com/oauth2/token",
	}
	httpClient := config.Client(oauth2.NoContext)
	twitterClient := twitter.NewClient(httpClient)

	return &Client{
		twitterClient:  twitterClient,
		tweetEntryChan: make(chan module.TweetEntry, 10),
	}, nil
}

func (c *Client) Chan() <- chan module.TweetEntry {
	return c.tweetEntryChan
}

func (c *Client) Close() {
	close(c.tweetEntryChan)
}

func (c *Client) GetTweetsForUser(username string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cacheDir := path.Join(cwd, "data", username)
	files, err := ioutil.ReadDir(cacheDir)
	if err != nil {
		err = os.MkdirAll(cacheDir, DEFAULT_PERMS)
		if err != nil {
			return err
		}
		files = []os.FileInfo{}
	}

	// reads contents of existing cache
	var maxCachedID int64
	for _, file := range files {
		contents, err := ioutil.ReadFile(path.Join(cacheDir, file.Name()))
		if err != nil {
			return err
		}

		var tweet twitter.Tweet
		err = json.Unmarshal(contents, &tweet)
		if err != nil {
			return err
		}

		c.tweetEntryChan <- module.TweetEntry{
			Tweet:    tweet,
			Username: username,
		}

		if maxCachedID < tweet.ID {
			maxCachedID = tweet.ID
		}
	}

	// fetches new tweets after the provided ID
	err = fetchTweets(c.twitterClient, c.tweetEntryChan, username, cacheDir, maxCachedID)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetTweetsForUsers(usernames []string) error {
	var wg sync.WaitGroup
	wg.Add(len(usernames))

	errChan := make(chan error)
	defer close(errChan)

	for _, username := range usernames {
		func(username string) {
			err := c.GetTweetsForUser(username)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
			}
			wg.Done()
		}(username)
	}

	wg.Wait()
	select {
	case err := <-errChan:
		return err
	default:
	}

	return nil
}

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
		defer func() { quitChan <- 0 }()

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
