# GoReddit

Reddit API client for [Go](http://golang.org)    
[LICENSE](LICENSE)

Currently, the following things are implemented:

* Logging in
* Getting submissions from a subreddit
* Getting comments for a thing
* Voting

GoReddit makes every effort to mirror the JSON output it receives from the Reddit API into native golang types and structs.

## Usage examples

### Creating a client

```go
import "goreddit" // https://github.com/kz26/GoReddit

client := goreddit.NewClient("MyBot/1.0")
```

Each client instance is thread-safe (can be called by multiple goroutines) and respects Reddit's guideline of no more than one request every 2 seconds.

### Getting submissions from a subreddit
```go

listing, err := rc.GetSubreddit("news", "new", 25) // Get new submissions from r/news, limit 25
for _, link := range listing {
	fmt.Println(link.Data.Title)
}
```

### Voting
```go
err := rc.Vote("t3_XXXXXX", 1) // Upvote the thing with the given ID
if err == nil {
	fmt.Println("Upvoted")
} else {
	fmt.Println("Error")
}
```