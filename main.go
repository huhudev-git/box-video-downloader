package main

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/huhugiter/box-video-downloader/models"

	flags "github.com/jessevdk/go-flags"
)

// Options contains the flag options
type Options struct {
	URL string `short:"i" description:"box input url"`
}

func main() {
	options := Options{}
	parser := flags.NewParser(&options, flags.Default)
	p, err := parser.Parse()
	if err != nil {
		if p == nil {
			fmt.Print(err)
		}
		return
	}

	// sharedName
	sharedNameReg := regexp.MustCompile(`/s/(.+)`)
	matchs := sharedNameReg.FindStringSubmatch(options.URL)
	if matchs == nil {
		return
	}
	sharedName := matchs[1]

	// cookies
	cookies, err := ioutil.ReadFile("./cookies")
	if len(cookies) == 0 {
		fmt.Println("Empty Cookies Provided")
		return
	}
	c := models.NewClient(string(cookies))

	// content
	content, err := c.GetContent(options.URL)
	if err != nil {
		panic(err)
	}

	// fileID
	fileID, err := c.GetFileID(string(content))
	if err != nil {
		panic(err)
	}
	fmt.Println("fileID:", fileID)

	// requestToken
	requestToken, err := c.GetRequestToken(string(content))
	if err != nil {
		panic(err)
	}
	//fmt.Println("requestToken:", requestToken)

	// tokens
	tokens, err := c.GetTokens(fileID, requestToken, sharedName)
	if tokens != nil && err != nil {
		panic(err)
	}
	//fmt.Println("tokens:", tokens)

	// info
	info, err := c.GetInfo(tokens.Write, fileID, sharedName)
	if err != nil {
		panic(err)
	}
	//fmt.Println("info:", info)
	fmt.Println("fileName:", info.Name)

	// manifest
	//manifest, err := c.GetManifest(tokens.Read, info.FileVersion.ID, fileID, sharedName)
	//if err != nil {
	//	fmt.Print(err)
	//	return
	//}
	//fmt.Println("manifest:", manifest)

	// download
	err = c.DownloadFile(tokens.Read, info.FileVersion.ID, info.Name, fileID, sharedName)
	if err != nil {
		panic(err)
	}
}
