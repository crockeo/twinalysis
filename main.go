package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dghubble/go-twitter/twitter"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/crockeo/twinalysis/module"
	"github.com/crockeo/twinalysis/module/averages"
	"github.com/crockeo/twinalysis/tweets"
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

	mod, err := getModule(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	tweetEntryChan := make(chan module.TweetEntry, 10)
	// TODO: Manage tweetEntryChan from main.go (instead of within CollectTweets)

	// TODO: Retrieve errors from CollectTweets
	go tweets.CollectTweets(client, tweetEntryChan, os.Args[2:])
	err = (*mod).AnalyzeTweets(tweetEntryChan)
	if err != nil {
		panic(err)
	}
}
