// Package downloader implements downloading from the osu! website, through,
// well, mostly scraping and dirty hacks.
package downloader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	"github.com/Gigamons/cheesegull/logger"
)

var downloadHostname = "old.ppy.sh"

// LogIn logs in into an osu! account and returns a Client.
func LogIn(username, password string, downloadhostname string) (*Client, error) {
	logger.Debug("Try to Login into Osu!")
	j, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		return nil, err
	}
	c := &http.Client{
		Jar: j,
	}
	vals := url.Values{}
	vals.Add("redirect", "/")
	vals.Add("sid", "")
	vals.Add("username", username)
	vals.Add("password", password)
	vals.Add("autologin", "on")
	vals.Add("login", "login")
	loginResp, err := c.PostForm("https://old.ppy.sh/forum/ucp.php?mode=login", vals)
	if err != nil {
		return nil, err
	}
	if loginResp.Request.URL.Path != "/" {
		return nil, errors.New("downloader: Login: could not log in (was not redirected to index)")
	}

	downloadHostname = downloadhostname
	if downloadhostname == "" {
		logger.Debug("WARNING! set downloadHostname to Default old.ppy.sh")
		downloadHostname = "old.ppy.sh"
	}

	return (*Client)(c), nil
}

// Client is a wrapper around an http.Client which can fetch beatmaps from the
// osu! website.
type Client http.Client

// HasVideo checks whether a beatmap has a video.
func (c *Client) HasVideo(setID int) (bool, error) {
	logger.Debug("Check if SetID %v has Video.", setID)
	h := (*http.Client)(c)

	page, err := h.Get(fmt.Sprintf("https://old.ppy.sh/s/%d", setID))
	if err != nil {
		logger.Debug("SetID %v don't has video!", setID)
		return false, err
	}
	defer page.Body.Close()
	body, err := ioutil.ReadAll(page.Body)
	if err != nil {
		logger.Debug("SetID %v has video!", setID)
		return false, err
	}
	return bytes.Contains(body, []byte(fmt.Sprintf(`href="/d/%dn"`, setID))), nil
}

// Download downloads a beatmap from the osu! website. noVideo specifies whether
// we should request the beatmap to not have the video.
func (c *Client) Download(setID int, noVideo bool) (io.ReadCloser, error) {
	suffix := ""
	if noVideo {
		suffix = "n"
	}

	logger.Debug("Download SetID %v. has video: %t", setID, noVideo)

	return c.getReader(strconv.Itoa(setID) + suffix)
}

// ErrNoRedirect is returned from Download when we were not redirect, thus
// indicating that the beatmap is unavailable.
var ErrNoRedirect = errors.New("no redirect happened, beatmap could not be downloaded")

func (c *Client) getReader(str string) (io.ReadCloser, error) {
	h := (*http.Client)(c)

	resp, err := h.Get(fmt.Sprintf("https://%s/d/", downloadHostname) + str)
	if err != nil {
		return nil, err
	}

	if resp.Request.URL.Host == "old.ppy.sh" {
		resp.Body.Close()
		return nil, ErrNoRedirect
	}

	x := bytes.NewBuffer(nil)

	io.Copy(x, resp.Body)

	z, err := ioutil.ReadAll(x)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(string(z), "<html>") {
		return nil, errors.New("Server down")
	}

	return resp.Body, nil
}
