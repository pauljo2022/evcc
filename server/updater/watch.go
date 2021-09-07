package updater

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/google/go-github/v32/github"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-version"
)

type webServer interface {
	Router() *mux.Router
}

type watch struct {
	log     api.Logger
	outChan chan<- util.Param
	repo    *Repo
}

func (u *watch) Send(key string, val interface{}) {
	u.outChan <- util.Param{
		Key: key,
		Val: val,
	}
}

func (u *watch) watchReleases(installed string, out chan *github.RepositoryRelease) {
	for range time.NewTicker(6 * time.Hour).C {
		rel, err := u.findReleaseUpdate(installed)
		if err != nil {
			u.log.Errorf("version check failed: %v", err)
			continue
		}

		if rel != nil {
			u.log.Infof("new version available: %s", *rel.TagName)
			out <- rel
		}
	}
}

// findReleaseUpdate validates if updates are available
func (u *watch) findReleaseUpdate(installed string) (*github.RepositoryRelease, error) {
	rel, err := u.repo.GetLatestRelease()
	if err != nil {
		return nil, err
	}

	if rel.TagName == nil {
		return nil, errors.New("untagged release")
	}

	v1, err := version.NewVersion(installed)
	if err != nil {
		return nil, err
	}

	v2, err := version.NewVersion(*rel.TagName)
	if err != nil {
		return nil, err
	}

	if v1.LessThan(v2) {
		go u.fetchReleaseNotes(installed)
		return rel, nil
	}

	// no update
	return nil, nil
}

// fetchReleaseNotes retrieves release notes up to semver and sends to client
func (u *watch) fetchReleaseNotes(installed string) {
	if notes, err := u.repo.ReleaseNotes(installed); err == nil {
		u.outChan <- util.Param{
			Key: "releaseNotes",
			Val: notes,
		}
	} else {
		u.log.Warnf("couldn't download release notes: %v", err)
	}
}
