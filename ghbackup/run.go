package ghbackup

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
)

func newExponentialBackOff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	return b
}

// Run update for the given Config.
func Run(config Config) error {
	// Defaults
	if config.Log == nil {
		config.Log = log.New(ioutil.Discard, "", 0)
	}
	if config.Err == nil {
		config.Err = log.New(ioutil.Discard, "", 0)
	}
	if config.Workers == 0 {
		config.Workers = defaultMaxWorkers
	}
	if config.API == "" {
		config.API = defaultAPI
	}
	if config.Doer == nil {
		config.Doer = http.DefaultClient
	}

	// Fetch list of repositories
	repos, err := fetch(config.Account, config.Secret, config.API, config.Doer)
	if err != nil {
		return err
	}

	config.Log.Printf("%d repositories:", len(repos))

	results := make(chan repoState)

	// Backup repositories in parallel with exponential-backoff retries
	go each(repos, config.Workers, func(r repo) {
		eBackoff := newExponentialBackOff()
		state, err := config.backup(r)
		for {
			if err != nil {
				sleepDuration := eBackoff.NextBackOff()
				if sleepDuration == backoff.Stop {
					config.Log.Printf("repository %v failed to get cloned: %v", r, err)
					break
				}
				config.Err.Println(err)
				time.Sleep(sleepDuration)
				state, err = config.backup(r)
				continue
			}
			break
		}
		results <- state
	})

	var creations, updates, unchanged, failed int

	for i := 0; i < len(repos); i++ {
		state := <-results
		if state == stateNew {
			creations++
		} else if state == stateChanged {
			updates++
		} else if state == stateUnchanged {
			unchanged++
		} else {
			failed++
		}
	}
	close(results)

	config.Log.Printf(
		"done: %d new, %d updated, %d unchanged",
		creations,
		updates,
		unchanged,
	)
	if failed > 0 {
		return fmt.Errorf("failed to get %d repositories", failed)
	}
	return nil
}

func each(repos []repo, workers int, worker func(repo)) {
	if len(repos) < workers {
		workers = len(repos)
	}

	jobs := make(chan repo)

	for w := 0; w < workers; w++ {
		go func() {
			for r := range jobs {
				worker(r)
			}
		}()
	}

	for _, r := range repos {
		jobs <- r
	}
	close(jobs)
}
