package gitmoji

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"
)

const (
	gitmojiURL string = "https://raw.githubusercontent.com/carloscuesta/gitmoji/master/packages/gitmojis/src/gitmojis.json"
	dirName    string = ".config/gogit"
	fileName   string = "gitmoji.json"
)

type (
	gitmojiCache struct {
		filePath string
		gitmoji  []Gitmoji
		url      string
	}

	Gitmoji struct {
		Code        string `json:"code"`
		Description string `json:"description"`
		Emoji       string `json:"emoji"`
		Entity      string `json:"entity"`
		Name        string `json:"name"`
	}

	fileData struct {
		gitmoji  []Gitmoji `json:"gitmoji"`
		modified time.Time `json:"modified"`
	}

	downloadData struct {
		gitmoji []Gitmoji `json:"gitmojis"`
	}
)

// GetGitmoji gets the gitmoji list from a local file cache if available;
// otherwise, downloads the latest gitmoji list from github.com.
func GetGitmoji() ([]Gitmoji, error) {
	gc, err := newGitmojiCache()
	if err != nil {
		return nil, err
	}

	res, err := ioutil.ReadFile(gc.filePath)
	if err != nil {
		err = gc.updateCache()
		if err != nil {
			return nil, err
		}
	}

	fd := new(fileData)
	err = json.Unmarshal(res, &fd)
	if err != nil {
		return nil, err
	}

	if isOlderThanThirtyDays(fd.modified) {
		err = gc.updateCache()
		if err != nil {
			return nil, err
		}
	}

	return fd.gitmoji, nil
}

func newGitmojiCache() (*gitmojiCache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %v", err)
	}

	filePath := path.Join(homeDir, dirName, fileName)

	return &gitmojiCache{
		filePath: filePath,
		url:      gitmojiURL,
		gitmoji:  nil,
	}, nil
}

func (gc gitmojiCache) updateCache() error {
	gitmoji, err := gc.download()
	if err != nil {
		return err
	}

	fd := fileData{
		gitmoji:  gitmoji,
		modified: time.Now(),
	}

	err = gc.writeCache(fd)
	if err != nil {
		return err
	}

	return nil
}

func (gc gitmojiCache) download() ([]Gitmoji, error) {
	fmt.Println("🌐  Fetching list of gitmoji...")

	r, err := http.Get(gc.url)
	if err != nil {
		return nil, fmt.Errorf("unable to download gitmoji list (from %s): %v", gc.url, err)
	}

	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to download gitmoji list (from %s): %v", gc.url, r.Status)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to download gitmoji list: %v", err)
	}

	gs := new(downloadData)

	err = json.Unmarshal(body, &gs)
	if err != nil {
		return nil, err
	}

	return gs.gitmoji, nil
}

func (gc *gitmojiCache) writeCache(data fileData) error {
	err := os.MkdirAll(path.Dir(gc.filePath), 0750)
	if err != nil {
		return fmt.Errorf("unable to create gitmoji cache directory: %v", err)
	}

	jb, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(gc.filePath, jb, 0600)
	if err != nil {
		return fmt.Errorf("unable to write gitmoji cache: %v", err)
	}

	return nil
}

func isOlderThanThirtyDays(timestamp time.Time) bool {
	duration := time.Since(timestamp)

	return duration.Hours() > 30*24
}
