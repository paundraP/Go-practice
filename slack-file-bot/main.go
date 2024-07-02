package main

import (
	"fmt"
	"os"

	"github.com/slack-go/slack"
)

func main() {
	// os.Setenv("bottoken", "your_bot_token")
	// os.Setenv("CHANNEL_ID", "your_channel_id")
	api := slack.New(os.Getenv("bottoken"))
	channelArr := os.Getenv("CHANNEL_ID")

	params := slack.UploadFileV2Parameters{
		File:     "/Users/paundrapujodarmawan/Code Here/go/practice/slack-file-bot/test.txt",
		Filename: "test",
		FileSize: 10,
		Channel:  channelArr,
	}
	file, err := api.UploadFileV2(params)

	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}

	fmt.Println(file.Title)
}
