package goreddit

import "encoding/json"
import "errors"
import "fmt"
import "io/ioutil"
import "net/http"
import "net/http/cookiejar"
import "net/url"
import "strconv"
import "time"

const REDDIT_HTTPS_URL = "https://ssl.reddit.com"
const REDDIT_URL = "http://www.reddit.com"
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
	return &Client{httpClient: &http.Client{Jar: jar}, UserAgent: userAgent, lock: make(chan bool, 1), lastAccess: time.Now().Add(-DELAY_S)}
}

// private utility function to do an API request
// implements rate-limiting and is thread-safe 
func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.lock <- true
	td := time.Now().Sub(c.lastAccess)
	if td < DELAY_S {
		time.Sleep(DELAY_S - td)
	}
	c.lastAccess = time.Now()
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return resp, err
	}
	<-c.lock
	return resp, err
}

// Struct representing the response from /api/login
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
func (c *Client) Login(user string, passwd string) error {
	params := make(url.Values)
	params.Set("api_type", "json")
	params.Set("user", user)
	params.Set("passwd", passwd)
	req, err := http.NewRequest("POST", fmt.Sprintf("%v/%v?%v", REDDIT_HTTPS_URL, "api/login", params.Encode()), nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	var data loginResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}
	if data.Json.Data.Modhash == "" {
		if len(data.Json.Errors) > 0 {
			return errors.New(fmt.Sprint(data.Json.Errors[0]))
		} else {
			return errors.New("Unknown login error")
		}
	} else {
		c.modhash = data.Json.Data.Modhash
	}
	return nil
}

// Fields for things that implement Votable
type Votable struct {
	Downs int
	Ups int
	Likes json.RawMessage // can't use bool here since Reddit also uses null; just check the raw string value if you need this field
}

// Fields for things that implement Created
type Created struct {
	Created float64
	Created_utc float64
}

// A comment
type Comment struct {
	Kind string
	Data struct {
		Votable
		Created

		Author string
		Body string
		Body_html string
		Id string
		Name string
		Permalink string
		Replies json.RawMessage
		Score int
		Title string
	}
}

// A link
type Link struct {
	Kind string
	Data struct {
		Votable
		Created

		Domain string
		Hidden bool
		Id string
		Is_self bool
		Name string
		Num_comments int
		Over_18 bool
		Permalink string
		Score int
		Title string
	}
}

// A listing of links
type LinkListing struct {
	Data struct {
		Modhash string
		Children []Link
		After string
		Before string
	}
	Kind string
}

// A listing of comments
type CommentListing struct {
	Data struct {
		Modhash string
		Children []Comment
		After string
		Before string
	}
	Kind string
}

// Get posts in a subreddit
// TODO: implement sorting
func (c *Client) GetSubreddit(sr string, sort string, limit int) ([]Link, error) {
	params := make(url.Values)
	params.Set("limit", strconv.Itoa(limit))
	req, err := http.NewRequest("GET", fmt.Sprintf("%v/r/%v/%v.json?%v", REDDIT_URL, sr, sort, params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var data LinkListing
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	return data.Data.Children, err
}


// Get comments for a specific thing ID
// TODO: implement sorting
func (c *Client) GetComments(id string, sort string, limit int) ([]Comment, error) {
	params := make(url.Values)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("sort", sort)
	req, err := http.NewRequest("GET", fmt.Sprintf("%v/comments/%v.json?%v", REDDIT_URL, id, params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var data []json.RawMessage
	err = json.Unmarshal(body, &data)
	if err != nil || len(data) != 2 {
		return nil, err
	}
	var cl CommentListing
	err = json.Unmarshal(data[1], &cl)
	return cl.Data.Children, err
}

func (com *Comment) GetReplies() ([]Comment, error) {
 	var replies CommentListing
 	err := json.Unmarshal(com.Data.Replies, &replies)
 		return replies.Data.Children, err
}

// Returns a flattened list of comments and replies
func GetCommentsFlat(comList []Comment) []Comment {
	cids := make([]Comment, len(comList))
	var fh func(cl []Comment)
	fh = func(cl []Comment) {
		for _, com := range cl {
			if com.Data.Author != "[deleted]" {
				cids = append(cids, com)
			}
			replies, err := com.GetReplies()
			if err == nil {
				if len(replies) > 0 {
					fh(replies)
				}
			}
		}
	}
	fh(comList)
	return cids
}

// Vote on a thing
// id: thing id, dir: 1, 0, -1 for upvote, null vote, and downvote, respectively
func (c *Client) Vote(id string, dir int) error {
	if c.modhash == "" {
		return errors.New("Login required")
	}
	params := make(url.Values)
	params.Set("id", id)
	params.Set("dir", string(dir))
	params.Set("uh", c.modhash)
	req, err := http.NewRequest("POST", fmt.Sprintf("%v/%v?%v", REDDIT_URL, "api/vote", params.Encode()), nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	if string(body) == "{}" {
		return nil
	} else {
		return errors.New("Vote failed")
	}
}
