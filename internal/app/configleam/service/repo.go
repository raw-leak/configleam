package service

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type repo struct {
	branch   string
	token    string
	url      string
	lastHash string
	dir      string
}

func (r repo) HasHashChanged() (string, bool, error) {
	reqURL, err := url.Parse(r.url)
	if err != nil {
		return "", false, err
	}

	reqURL.User = url.UserPassword("username", r.token)

	cmd := exec.Command("git", "ls-remote", r.url, r.branch)

	output, err := cmd.Output()
	if err != nil {
		return "", false, err
	}
	if len(output) < 1 {
		return "", false, fmt.Errorf("brach '%s' has not been detected", r.branch)
	}

	log.Println(string(output))

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	hash := ""

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		parts := strings.Split(line, "\t")
		fmt.Println(parts)

		if len(parts) == 2 && parts[1] == "refs/heads/"+r.branch {
			hash = parts[0]
			break
		}
	}

	return hash, hash != r.lastHash, nil
}

func (r *repo) SetLastHash(hash string) {
	r.lastHash = hash
}

func (r *repo) CloneRemoteRepo() error {
	auth := &http.BasicAuth{
		Username: "username",
		Password: r.token,
	}

	_, err := git.PlainClone(r.dir, false, &git.CloneOptions{
		URL:           r.url,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", r.branch)),
		SingleBranch:  true,
		Auth:          auth,
	})

	return err
}

func (r repo) RemoveLocalRepo() error {
	err := os.RemoveAll(r.dir)
	if err != nil {
		return err
	}

	return nil
}

func (r *repo) pullLocalRepository() error {
	localRepo, err := git.PlainOpen(r.dir)
	if err != nil {
		return err
	}

	w, err := localRepo.Worktree()
	if err != nil {
		return err
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	return nil
}

func (r *repo) GetLatestConfig() (bool, error) {

	return true, nil
}

func newRepo(url, branch, token string) (*repo, error) {
	dir, err := getRepoNameFromURL(url)
	if err != nil {
		return nil, err
	}

	return &repo{url: url, branch: branch, token: token, dir: "repositories/" + dir}, nil
}

func getRepoNameFromURL(repoURL string) (string, error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", err
	}

	parts := strings.Split(parsedURL.Path, "/")
	if len(parts) > 0 {
		return strings.TrimSuffix(parts[len(parts)-1], ".git"), nil
	}

	return "", fmt.Errorf("could not extract repository name from URL")
}
