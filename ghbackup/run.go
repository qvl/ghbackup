package ghbackup

import (
	"fmt"
	"net/http"
	"sync"
)

// Run update for the given Config.
func Run(config Config) error {
	// Defaults
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
	config.Updates <- Update{UInfo, fmt.Sprintf("Backup %d repositories", len(repos))}

	// Backup repositories in parallel
	each(repos, config.Workers, func(r repo) {
		err := backup(config.Dir, config.Account, r, config.Updates)
		if err != nil {
			config.Updates <- Update{UErr, err.Error()}
		}
	})

	return nil
}

func each(repos []repo, workers int, worker func(repo)) {
	if len(repos) < workers {
		workers = len(repos)
	}

	jobs := make(chan repo)

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for r := range jobs {
				worker(r)
			}
		}()
	}

	for _, r := range repos {
		jobs <- r
	}
	close(jobs)
	wg.Wait()
}
