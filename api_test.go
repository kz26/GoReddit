package goreddit

import "fmt"
import "testing"

const TEST_USER_AGENT = "GoReddit-TestSuite"

func TestLogin(t *testing.T) {
	t.Skip("Login test skipped")
	client := NewClient(TEST_USER_AGENT)
	err := client.Login("user", "password")
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("Login successful")
		fmt.Println("modhash: ", client.modhash)
	}
}

func TestGetSubreddit(t *testing.T) {
	client := NewClient(TEST_USER_AGENT)
	listing, err := client.GetSubreddit("news", "new", 5)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(listing)
	}
}


func TestGetComments(t *testing.T) {
	client := NewClient(TEST_USER_AGENT)
	srListing, err := client.GetSubreddit("news", "hot", 1)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	fmt.Println()
	firstLinkID := srListing[0].Data.Id
	fmt.Println(firstLinkID)
	comments, err := client.GetComments(firstLinkID, "hot", 2)
	if err != nil {
		t.Error(err)
	} else {
		fc, _ := comments[0].GetReplies()
		fmt.Println(fc[0].Data.Body)
	}
}
