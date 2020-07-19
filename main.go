package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dghubble/go-twitter/twitter"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/crockeo/twinalysis/module"
	"github.com/crockeo/twinalysis/module/averages"
)

const (
	API_KEY_FILE        string = "secrets/api"
	API_SECRET_KEY_FILE string = "secrets/api_secret"
	BEARER_TOKEN_FILE   string = "secrets/bearer"

	DEFAULT_PERMS os.FileMode = 0755
)

var MODULES = []module.Module{averages.Averages{}}

func readKey(path string) (string, error) {
	rawKey, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(rawKey), nil
}

func twitterClient() (*twitter.Client, error) {
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
	client := twitter.NewClient(httpClient)

	return client, nil
}

// getAllTweets retrieves all tweets from a user and puts them through a channel. Once all of the
// tweets have been retrieved, it closes the channel.
func getAllTweets(tweetChannel chan twitter.Tweet, client *twitter.Client, user string) ([]twitter.Tweet, error) {
	excludeReplies := false
	includeRetweets := false

	tweets := []twitter.Tweet{}
	var maxID int64 = 1
	for {
		tweetBatch, _, err := client.Timelines.UserTimeline(
			&twitter.UserTimelineParams{
				ScreenName:      user,
				MaxID:           maxID - 1, // Only show new tweets
				ExcludeReplies:  &excludeReplies,
				IncludeRetweets: &includeRetweets,
				TweetMode:       "extended",
			},
		)
		if err != nil {
			return []twitter.Tweet{}, err
		}
		if len(tweetBatch) == 0 {
			break
		}

		var tweet twitter.Tweet
		for _, tweet = range tweetBatch {
			tweets = append(tweets, tweet)
			tweetChannel <- tweet
		}

		maxID = tweet.ID
	}

	return tweets, nil
}

// saveTweet saves a tweet to a location on disk.
func saveTweet(tweet twitter.Tweet, path string) error {
	data, err := json.Marshal(tweet)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(
		path,
		data,
		DEFAULT_PERMS,
	)
}

// saveTweets processes tweets from a channel and saves them to disk based on their user name and
// ID.
func saveTweets(tweetChannel chan twitter.Tweet, tweetDir string) error {
	pathTemplate := "%s/%d.json"
	for tweet := range tweetChannel {
		err := saveTweet(tweet, fmt.Sprintf(pathTemplate, tweetDir, tweet.ID))
		if err != nil {
			return err
		}
	}
	return nil
}

// loadTweet loads a tweet from a location on disk.
func loadTweet(path string) (twitter.Tweet, error) {
	tweet := twitter.Tweet{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return tweet, err
	}

	err = json.Unmarshal(data, &tweet)
	if err != nil {
		return tweet, err
	}

	return tweet, nil
}

// collectTweets collects all tweets of the provided user, either from online or from a local cache,
// and returns them for arbitrary usage.
func collectTweets(client *twitter.Client, user string) ([]twitter.Tweet, error) {
	cwd, err := os.Getwd()
	if err != err {
		return []twitter.Tweet{}, err
	}

	tweetDir := fmt.Sprintf(
		"%s/data/%s",
		cwd,
		user,
	)

	if _, err := os.Stat(tweetDir); os.IsNotExist(err) {
		err = os.MkdirAll(tweetDir, DEFAULT_PERMS)
		if err != nil {
			return []twitter.Tweet{}, err
		}

		tweetChannel := make(chan twitter.Tweet, 10)
		go saveTweets(tweetChannel, tweetDir)
		tweets, err := getAllTweets(tweetChannel, client, user)
		if err != nil {
			return []twitter.Tweet{}, err
		}
		close(tweetChannel)

		return tweets, nil
	}

	files, err := ioutil.ReadDir(tweetDir)
	if err != nil {
		return []twitter.Tweet{}, err
	}

	tweets := make([]twitter.Tweet, len(files))
	for i, file := range files {
		tweet, err := loadTweet(fmt.Sprintf("%s/%s", tweetDir, file.Name()))
		if err != nil {
			return []twitter.Tweet{}, err
		}
		tweets[i] = tweet
	}

	return tweets, nil
}

func getModule(name string) (*module.Module, error) {
	for _, module := range MODULES {
		if module.Name() == name {
			return &module, nil
		}
	}
	return nil, fmt.Errorf("No such module '%s'", name)
}

func main() {
	client, err := twitterClient()
	if err != nil {
		panic(err)
	}

	if len(os.Args) < 3 {
		fmt.Println("Insufficient arguments")
		return
	}

	module, err := getModule(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	tweetsByUsername := map[string][]twitter.Tweet{}
	for _, username := range os.Args[2:] {
		tweets, err := collectTweets(client, username)
		if err != nil {
			fmt.Println(err)
			return
		}

		tweetsByUsername[username] = tweets
	}

	err = (*module).AnalyzeTweets(tweetsByUsername)
	if err != nil {
		fmt.Println(err)
		return
	}

}
