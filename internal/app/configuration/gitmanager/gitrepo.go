package gitmanager

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/raw-leak/configleam/internal/app/configuration/helper"
	"golang.org/x/net/context"
)

// TODO:

// 4. Function Decomposition
// Large Functions: Some functions like ExtractConfigListFromLocalRepo are doing quite a lot. Consider breaking them down into smaller, more focused functions. This makes your code easier to understand, test, and maintain.

// 5. Handling Different File Types
// Extensibility for File Types: Currently, the code explicitly checks for .yaml and .yml extensions. If you plan to support more file types in the future, consider designing this part to be more extensible.

// 9. Concurrency Considerations
// If your application will handle multiple Git repositories simultaneously, consider the implications for concurrency. Ensure that operations like cloning, pulling updates, and extracting configurations are safe to run in parallel.

// 6. Security Considerations
// Hardcoded Credentials: The use of a hardcoded username in CloneRemoteRepo and PullUpdatesFromRemoteRepo (Username: "username") is not ideal. Consider making this configurable or using a more secure method of authentication.

// TODO: change name of Env
type Env struct {
	Name    string
	LastTag string
	SemVer  helper.SemanticVersion
}

// GitRepository represents a Git repository for managing configurations.
type GitRepository struct {
	URL      string
	Branch   string
	Name     string
	Dir      string
	LastHash string
	LastTag  string
	Envs     map[string]Env
	Mux      sync.RWMutex

	locRep *git.Repository
	wt     *git.Worktree
}

// NewGitRepository creates and initializes a new GitRepository instance.
func NewGitRepository(repoURL, branch string, envs []string) (*GitRepository, error) {
	repoName, err := helper.ExtractRepoNameFromRepoURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("error extracting repo name: %w", err)
	}

	envsParam := map[string]Env{}
	for _, env := range envs {
		envsParam[env] = Env{Name: env}
	}

	repoDir := filepath.Join("repositories", repoName)
	return &GitRepository{URL: repoURL, Branch: branch, Dir: repoDir, Envs: envsParam, Name: repoName}, nil
}

func (gr *GitRepository) getAuth() *http.BasicAuth {
	return &http.BasicAuth{
		Username: "username",
		Password: os.Getenv("GIT_ACCESS_TOKEN"),
	}
}

// CloneRemoteRepo clones the remote repository.
func (gr *GitRepository) CloneRemoteRepo() error {
	gr.Mux.Lock()
	defer gr.Mux.Unlock()

	var err error
	gr.locRep, err = git.PlainClone(gr.Dir, false, &git.CloneOptions{
		URL:           gr.URL,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", gr.Branch)),
		SingleBranch:  true,
		Auth:          gr.getAuth(),
	})
	if err != nil {
		return fmt.Errorf("error cloning repository: %w", err)
	}

	gr.wt, err = gr.locRep.Worktree()
	if err != nil {
		return fmt.Errorf("error getting worktree: %w", err)
	}

	return nil
}

func (gr *GitRepository) FetchAndCheckout(tag string) error {
	gr.Mux.Lock()
	defer gr.Mux.Unlock()

	err := gr.locRep.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("refs/tags/%s:refs/tags/%s", tag, tag))},
		Auth:       gr.getAuth(),
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("error fetching tag '%s': %w", tag, err)
	}

	err = gr.wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", tag)),
		Force:  true,
	})
	if err != nil {
		return fmt.Errorf("error checking out tag '%s': %w", tag, err)
	}

	return nil
}

// RemoveLocalRepo removes the local repository directory.
func (gr *GitRepository) RemoveLocalRepo() error {
	gr.Mux.Lock()
	defer gr.Mux.Unlock()

	if err := os.RemoveAll(gr.Dir); err != nil {
		return fmt.Errorf("error removing local repository: %w", err)
	}

	gr.locRep = nil
	gr.wt = nil

	return nil
}

// PullUpdatesFromRemoteRepo pulls updates from the remote repository.
func (gr *GitRepository) PullUpdatesFromRemoteRepo() error {
	gr.Mux.Lock()
	defer gr.Mux.Unlock()

	err := gr.wt.Pull(&git.PullOptions{RemoteName: "origin", Auth: gr.getAuth()})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Println(err)
		return fmt.Errorf("error pulling updates: %w", err)
	}

	if err != nil {
		return err
	}

	return nil
}

func (gr *GitRepository) PullTagsFromRemoteRepo() ([]string, error) {
	gr.Mux.Lock()
	defer gr.Mux.Unlock()

	err := gr.locRep.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{"refs/tags/*:refs/tags/*"},
		Tags:       git.AllTags,
		Auth:       gr.getAuth(),
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("error fetching tags: %w", err)
	}

	repoTags, err := gr.locRep.Tags()
	if err != nil {
		return nil, fmt.Errorf("error listing tags: %w", err)
	}
	defer repoTags.Close()

	tags := []string{}
	err = repoTags.ForEach(func(t *plumbing.Reference) error {
		tags = append(tags, t.Name().Short())
		return nil
	})

	if err != nil {
		return nil, err
	}

	return tags, nil
}

func (gr *GitRepository) SetEnvLatestVersion(_ context.Context, env string, lastTag string, lastSemVer helper.SemanticVersion) error {
	gitEnv, ok := gr.Envs[env]
	if !ok {
		return fmt.Errorf("error while setting new tag and version for environment '%s'", env)
	}

	gitEnv.LastTag = lastTag
	gitEnv.SemVer = lastSemVer

	gr.Envs[env] = gitEnv

	return nil
}
