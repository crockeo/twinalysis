package main

import (
	"fmt"
	"os"

	"github.com/crockeo/twinalysis/module"
	"github.com/crockeo/twinalysis/module/averages"
	"github.com/crockeo/twinalysis/tweets"
)

const (
	DEFAULT_PERMS os.FileMode = 0755
)

var MODULES = []module.Module{averages.Averages{}}

func getModule(name string) (*module.Module, error) {
	for _, module := range MODULES {
		if module.Name() == name {
			return &module, nil
		}
	}
	return nil, fmt.Errorf("No such module '%s'", name)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Insufficient arguments")
		return
	}

	mod, err := getModule(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	client, err := tweets.NewClient()
	if err != nil {
		panic(err)
	}

	errChan := make(chan error)
	defer close(errChan)

	go (*mod).AnalyzeTweets(client.Chan(), errChan)
	err = client.GetTweetsForUsers(os.Args[2:])
	if err != nil {
		panic(err)
	}

	for {
		if len(client.Chan()) == 0 {
			break
		}
	}
	client.Close()

	err = <-errChan
	if err != nil {
		panic(err)
	}
}
