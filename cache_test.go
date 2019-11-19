package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
)

func TestInvalidJSON(t *testing.T) {
	f, err := ioutil.TempFile("", "gitmoji")
	defer os.Remove(f.Name())

	if err != nil {
		t.Fatal(err)
	}

	_, err = f.Write([]byte("Invalid JSON"))

	if err != nil {
		t.Fatal(err)
	}

	err = f.Close()

	if err != nil {
		t.Fatal(err)
	}

	cache := GitmojiCache{
		CacheFile: f.Name(),
		gitmoji:   nil,

		load: func() ([]byte, error) {
			return []byte("This is also not valid JSON"), nil
		},
	}

	// Load from cache file
	_, err = cache.GetGitmoji()

	if err == nil {
		t.Fatal("Expected error getting gitmoji from cache file")
	}

	err = os.Remove(cache.CacheFile)

	if err != nil {
		t.Fatal(err)
	}

	// Load via load function
	_, err = cache.GetGitmoji()

	if err == nil {
		t.Fatal("Expected error getting gitmoji from loader")
	}
}

func TestEmptyJSON(t *testing.T) {
	f, err := ioutil.TempFile("", "gitmoji")
	defer os.Remove(f.Name())

	if err != nil {
		t.Fatal(err)
	}

	_, err = f.Write([]byte("{}"))

	if err != nil {
		t.Fatal(err)
	}

	err = f.Close()

	if err != nil {
		t.Fatal(err)
	}

	cache := GitmojiCache{
		CacheFile: f.Name(),
		gitmoji:   nil,

		load: func() ([]byte, error) {
			t.Fatal("This should not be called")
			return nil, nil
		},
	}

	// Load from cache file
	gitmoji, err := cache.GetGitmoji()

	if err != nil {
		t.Fatal("Expected empty gitmoji list, got error: ", err)
	}

	if len(gitmoji) != 0 {
		t.Fatal("Expected empty gitmoji list, got this: ", gitmoji)
	}
}

func TestUnreadableCacheFile(t *testing.T) {
	f, err := ioutil.TempFile("", "gitmoji")
	defer os.Remove(f.Name())

	if err != nil {
		t.Fatal(err)
	}

	_, err = f.Write([]byte("{}"))

	if err != nil {
		t.Fatal(err)
	}

	err = f.Close()

	if err != nil {
		t.Fatal(err)
	}

	cache := GitmojiCache{
		CacheFile: f.Name(),
		gitmoji:   nil,

		load: func() ([]byte, error) {
			t.Fatal("This should not be called")
			return nil, nil
		},
	}

	err = os.Chmod(f.Name(), 0)

	if err != nil {
		t.Fatal("Unable to make file unreadable: ", err)
	}

	// Load from cache file
	_, err = cache.GetGitmoji()

	if err == nil {
		t.Fatal("Expected error reading cache file.")
	}
}

func TestErrorFetchingData(t *testing.T) {
	cacheFile := path.Join(os.TempDir(), "gitmoji-file-not-found.json")
	cache := GitmojiCache{
		CacheFile: cacheFile,
		gitmoji:   nil,

		load: func() ([]byte, error) {
			return nil, fmt.Errorf("trigger an error")
		},
	}

	// Load from cache file
	_, err := cache.GetGitmoji()

	if err == nil {
		t.Fatal("Expected error fetching data.")
	}
}

func TestLoadFromURL(t *testing.T) {
	cacheFile := path.Join(os.TempDir(), "gitmoji-temp-file.json")

	// Testing HTTP Server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{
			"gitmojis": [
			  {
				"emoji": "🎨 ",
				"entity": "&#x1f3a8;",
				"code": ":art:",
				"description": "Improving structure / format of the code.",
				"name": "art"
			  }
			]
		}`))
	}))
	defer server.Close()

	// Load emoji...
	cache, err := NewGitmojiCacheWithURLAndCacheFile(server.URL, cacheFile)

	// Load from cache file
	gitmoji, err := cache.GetGitmoji()

	if err != nil {
		os.Remove(cacheFile)
		t.Fatal(err)
	}

	if len(gitmoji) != 1 {
		os.Remove(cacheFile)
		t.Fatal("Didn't read gitmoji correctly; read this instead: ", gitmoji)
	}

	err = os.Remove(cacheFile)

	if err != nil {
		t.Fatal(err)
	}
}

func TestLoad404(t *testing.T) {
	cacheFile := path.Join(os.TempDir(), "gitmoji-temp-file.json")

	err := os.Remove(cacheFile)

	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	// Testing HTTP Server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Load emoji...
	cache, err := NewGitmojiCacheWithURLAndCacheFile(server.URL, cacheFile)

	// Load from cache file
	_, err = cache.GetGitmoji()

	if err == nil {
		os.Remove(cacheFile)
		t.Fatal("Expected error fetching gitmoji list.")
	}

	if !strings.Contains(err.Error(), "404") {
		os.Remove(cacheFile)
		t.Fatal("Expected a 404 error; got: ", err)
	}

	err = os.Remove(cacheFile)

	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
}