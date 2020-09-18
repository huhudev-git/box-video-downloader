package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/huhugiter/box-video-downloader/models"

	flags "github.com/jessevdk/go-flags"
)

// Options contains the flag options
type Options struct {
	URL    string `short:"i" description:"box input url"`
	Docker bool   `short:"d" description:"ffmpeg use docker"`
}

func main() {
	options := Options{}
	parser := flags.NewParser(&options, flags.Default)
	p, err := parser.Parse()
	if err != nil {
		if p == nil {
			panic(err)
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
	cookiesPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// exist
	cookiesPath = path.Join(cookiesPath, "cookies")
	_, err = os.Stat(cookiesPath)
	if os.IsNotExist(err) {
		panic(err)
	}

	// read
	cookies, err := ioutil.ReadFile(cookiesPath)
	if len(cookies) == 0 {
		fmt.Println("Empty Cookies Provided")
		return
	}
	cookiesStr := strings.Replace(string(cookies), "\n", "", -1)
	c := models.NewClient(cookiesStr)

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
	if len(fileID) == 0 {
		panic("Expired Cookies")
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
	manifest, err := c.GetManifest(tokens.Read, info.FileVersion.ID, fileID, sharedName)
	if err != nil {
		fmt.Print(err)
		return
	}

	// resolution
	re := regexp.MustCompile(`initialization="video/(\d+?)/init.m4s"`)
	resolutions := re.FindStringSubmatch(manifest)
	if resolutions == nil {
		fmt.Println("Get resolution failed")
	}
	resolution := resolutions[1]
	//fmt.Println("manifest:", manifest)

	// download
	err = c.DownloadFile(tokens.Read, info.FileVersion.ID, info.Name, fileID, sharedName, resolution, options.Docker)
	fmt.Println(err)
	if err != nil {
		panic(err)
	}
}
