package goreddit

import "encoding/json"
import "errors"
import "fmt"
import "io/ioutil"
import "net/http"
import "net/http/cookiejar"
import "net/url"
import "time"

const REDDIT_URL = "https://ssl.reddit.com"
const DELAY_S = 2 * time.Second

// Client represents a custom Reddit client that respects the Reddit API rate limit guidelines
type Client struct {
	httpClient *http.Client
	modhash string
	UserAgent string
	lock chan bool
	lastAccess time.Time
}

// Create a new client with the given user agent
func NewClient(userAgent string) *Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil
	}
	return &Client{httpClient: &http.Client{Jar: jar}, UserAgent: userAgent, lock: make(chan bool, 1), lastAccess: time.Now().Add(-DELAY_S),}
}

// private utility function to do an API request
// implements rate-limiting and is thread-safe 
func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.lock <- true
	td := time.Now().Sub(c.lastAccess)
	if td < DELAY_S {
		time.Sleep(DELAY_S - td)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return resp, err
	}
	<-c.lock
	return resp, err
}

// Struct representing the resposne from /api/login
type loginResponse struct {
	Json struct {
		Errors [][]string
		Data struct {
			Modhash string
			Cookie string
		}
	}
}

// Perform a login using the given username and password
func (c *Client) Login(user string, passwd string) (bool, error) {
	params := make(url.Values)
	params.Set("api_type", "json")
	params.Set("user", user)
	params.Set("passwd", passwd)
	req, err := http.NewRequest("POST", fmt.Sprintf("%v/%v?%v", REDDIT_URL, "api/login", params.Encode()), nil)
	if err != nil {
		return false, err
	}
	resp, err := c.do(req)
	if err != nil {
		return false, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	var data loginResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return false, err
	}
	if data.Json.Data.Modhash == "" {
		if len(data.Json.Errors) > 0 {
			return false, errors.New(fmt.Sprint(data.Json.Errors[0]))
		} else {
			return false, errors.New("Unknown login error")
		}
	} else {
		c.modhash = data.Json.Data.Modhash
	}
	return true, nil
}

// Recursive data structure for "things" on Reddit
type Thing struct {
	Kind string
	Data struct {
		After string
		Before string
		Children []Thing
		Downs int
		Id string
		Modhash string
		Name string
		Permalink string
		Replies interface{} // Reddit returns "" for empty but {} for non-empty, so we need to accomodate both
		Score int
		Title string
		Ups int
	}
}

// Utility function that performs a request that returns a thing
func (c *Client) getThing(req *http.Request) (*Thing, error) {
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var data Thing
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	return &data, err
}

// Utility function that performs a request that returns a JSON array of things
func (c *Client) getThings(req *http.Request) (*[]Thing, error) {
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var data []Thing
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	return &data, err
}

// Get posts in a subreddit
// TODO: implement sorting
func (c *Client) GetSubreddit(sr string, sort string, limit int) (*Thing, error) {
	params := make(url.Values)
	params.Set("limit", string(limit))
	req, err := http.NewRequest("GET", fmt.Sprintf("%v/r/%v.json?%v", REDDIT_URL, sr, params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	return c.getThing(req)
}

// Get comments for a specific thing ID
// TODO: implement sorting
func (c *Client) GetComments(id string, sort string, limit int) (*[]Thing, error) {
	params := make(url.Values)
	params.Set("limit", string(limit))
	req, err := http.NewRequest("GET", fmt.Sprintf("%v/comments/%v.json?%v", REDDIT_URL, id, params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	return c.getThings(req)
}

// Vote on a thing
// id: thing id, dir: 1, 0, -1 for upvote, null vote, and downvote, respectively
func (c *Client) Vote(id string, dir int) (bool, error) {
	if c.modhash == "" {
		return false, errors.New("Login required")
	}
	params := make(url.Values)
	params.Set("id", id)
	params.Set("dir", string(dir))
	params.Set("uh", c.modhash)
	req, err := http.NewRequest("POST", fmt.Sprintf("%v/%v?%v", REDDIT_URL, "api/vote", params.Encode()), nil)
	if err != nil {
		return false, err
	}
	resp, err := c.do(req)
	if err != nil {
		return false, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return false, err
	}
	if string(body) == "{}" {
		return true, nil
	} else {
		return false, errors.New("Vote failed")
	}
}
