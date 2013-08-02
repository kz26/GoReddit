package goreddit

import "fmt"
import "testing"

const TEST_USER_AGENT = ""

func TestLogin(t *testing.T) {
	client := NewClient("Test/1.0")
	t.Skip("Login test skipped")
	_, err := client.Login("user", "password")
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("Login successful")
		fmt.Println("modhash: ", client.modhash)
	}
}

func TestGetSubreddit(t *testing.T) {
	client := NewClient("Test2/1.0")
	listing, err := client.GetSubreddit("news", "hot", 2)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(listing)
	}
}

func TestGetComments(t *testing.T) {
	client := NewClient("Test2/1.0")
	srListing, err := client.GetSubreddit("news", "hot", 2)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	firstLinkID := srListing.Data.Children[0].Data.Id
	fmt.Println(firstLinkID)
	comments, err := client.GetComments(firstLinkID, "hot", 25)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(comments)
	}
}
