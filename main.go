package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/huhugiter/box-video-downloader/models"
	"github.com/levigross/grequests"

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
			fmt.Println(err)
			panic(err)
		}
		return
	}

	// cookies
	cookiesPath, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	// exist
	cookiesPath = path.Join(cookiesPath, "cookies")
	_, err = os.Stat(cookiesPath)
	if os.IsNotExist(err) {
		fmt.Println(err)
		panic(err)
	}

	// read
	cookies, err := ioutil.ReadFile(cookiesPath)
	if len(cookies) == 0 {
		fmt.Println("Empty Cookies Provided")
		return
	}
	cookiesStr := strings.Replace(string(cookies), "\n", "", -1)

	cookieJar, _ := cookiejar.New(nil)
	cs := []*http.Cookie{}
	for _, c := range strings.Split(cookiesStr, ";") {
		nv := strings.Split(c, "=")
		cs = append(cs, &http.Cookie{
			Name:     nv[0],
			Value:    nv[1],
			Path:     "/",
			MaxAge:   0,
			Domain:   "tus.app.box.com",
			Secure:   true,
			HttpOnly: false,
		})
	}
	cookieURL, _ := url.Parse("https://tus.app.box.com/")
	cookieJar.SetCookies(cookieURL, cs)

	session := &grequests.Session{
		HTTPClient: &http.Client{
			Jar: cookieJar,
		},
		RequestOptions: &grequests.RequestOptions{},
	}
	c := models.NewClient(session)

	urls := strings.Split(options.URL, ",")
	for _, url := range urls {
		// content
		content, err := c.GetContent(url)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		// fileID
		fileID, err := c.GetFileID(content)
		if err != nil {
			panic(err)
		}
		if len(fileID) == 0 {
			panic("Expired Cookies")
		}
		fmt.Println("FileID:", fileID)

		// requestToken
		requestToken, err := c.GetRequestToken(content)
		if err != nil {
			panic(err)
		}
		//fmt.Println("requestToken:", requestToken)

		// sharedName
		sharedNameReg := regexp.MustCompile(`/s/(.+)`)
		matchs := sharedNameReg.FindStringSubmatch(url)
		if matchs == nil {
			return
		}
		sharedName := matchs[1]

		// tokens
		tokens, err := c.GetTokens(fileID, requestToken, sharedName)
		if tokens != nil && err != nil {
			fmt.Println(err)
			panic(err)
		}
		//fmt.Println("tokens:", tokens)

		// info
		info, err := c.GetInfo(tokens.Write, fileID, sharedName)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		//fmt.Println("info:", info)
		fmt.Println("Filename:", info.Name)

		// manifest
		manifest, err := c.GetManifest(tokens.Read, info.FileVersion.ID, fileID, sharedName)
		if err != nil {
			panic(err)
		}

		// mediaPresentationDuration
		mediaPresentationDurationReg := regexp.MustCompile(`mediaPresentationDuration="(PT.+?)"`)
		matchs = mediaPresentationDurationReg.FindStringSubmatch(manifest)
		if matchs == nil {
			return
		}
		mediaPresentationDuration := matchs[1]
		duration := convert2second(mediaPresentationDuration)
		fmt.Println("Duration:", duration)

		// resolution
		resolutionReg := regexp.MustCompile(`initialization="video/(\d+?)/init.m4s"`)
		resolutions := resolutionReg.FindStringSubmatch(manifest)
		if resolutions == nil {
			fmt.Println("Get resolution failed")
			return
		}
		resolution := resolutions[1]
		fmt.Println("Resolution:", resolution+"P")

		// download
		err = c.DownloadFile(
			tokens.Read,
			info.FileVersion.ID,
			info.Name,
			fileID,
			sharedName,
			resolution,
			int(math.Ceil(duration/5.0))+5,
			1, // larger than 1 may loss data, unknown reason, may be server limit
			options.Docker,
		)

		if err != nil {
			panic(err)
		}
	}
}

func convert2second(mediaPresentationDuration string) float64 {
	var reg *regexp.Regexp
	if strings.Contains(mediaPresentationDuration, "D") {
		reg = regexp.MustCompile(`PT(.+?)D(.+?)H(.+?)M(.+?)\.(.+?)S`)
	} else if strings.Contains(mediaPresentationDuration, "H") {
		reg = regexp.MustCompile(`PT(.+?)H(.+?)M(.+?)\.(.+?)S`)
	} else if strings.Contains(mediaPresentationDuration, "M") {
		reg = regexp.MustCompile(`PT(.+?)M(.+?)\.(.+?)S`)
	} else {
		reg = regexp.MustCompile(`PT(.+?)\.(.+?)S`)
	}

	matchs := reg.FindStringSubmatch(mediaPresentationDuration)
	if matchs == nil {
		return 0
	}

	t := 0.0
	table := []float64{3600 * 24, 3600, 60, 1, 0.01}
	offect := 5 - len(matchs)
	for i := len(matchs) - 1; i > 0; i-- {
		o, _ := strconv.Atoi(matchs[i])
		t += float64(o) * table[offect+i]
	}
	return t
}
