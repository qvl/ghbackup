package ghbackup

import (
	"io/ioutil"
	"log"
	"net/http"
)

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

	results := make(chan struct {
		count int
		new   bool
	})

	// Backup repositories in parallel
	go each(repos, config.Workers, func(r repo) {
		objectCount, isNew, err := config.backup(r)
		if err != nil {
			config.Err.Println(err)
		}
		results <- struct {
			count int
			new   bool
		}{count: objectCount, new: isNew}
	})

	var creations, updates, count int

	for i := 0; i < len(repos); i++ {
		r := <-results
		if r.count > 0 {
			if r.new {
				creations++
			} else {
				updates++
			}
		}
		count += r.count
	}
	close(results)

	config.Log.Printf(
		"done: %d new, %d updated, %d unchanged, %d total objects\n",
		creations,
		updates,
		len(repos)-creations-updates,
		count,
	)

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
