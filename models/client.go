package models

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"sync"

	"github.com/levigross/grequests"
)

var waitGroutp = sync.WaitGroup{}

// Client box client
type Client struct {
	session *grequests.Session
}

// NewClient creates a new client
func NewClient(session *grequests.Session) *Client {
	c := &Client{
		session: session,
	}
	return c
}

// Login -
func (c *Client) Login() {
	session := grequests.NewSession(&grequests.RequestOptions{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36 Edg/86.0.622.51",
		Headers: map[string]string{
			"Connection": "keep-alive",
			"Accept":     "*/*",
			"Host":       "account.box.com",
			"Origin":     "https://account.box.com",
			"Referer":    "https://account.box.com/login",
		},
	})

	resp, err := session.Get("https://tus.account.box.com/login", nil)
	if err != nil {
		panic(err)
	}

	resp, err = session.Get("https://account.box.com/login", nil)
	if err != nil {
		panic(err)
	}

	re := regexp.MustCompile(`name="request_token" value="(.+?)"`)
	var requestToken string
	if m := re.FindStringSubmatch(resp.String()); m != nil {
		requestToken = m[1]
	}

	resp, err = session.Post(
		"https://account.box.com/login?redirect_url=%2F",
		&grequests.RequestOptions{
			Data: map[string]string{
				"login":             "@ed.tus.ac.jp",
				"_pw_sql":           "",
				"dummy-password":    "",
				"request_token":     requestToken,
				"redirect_url":      "/",
				"login_page_source": "email-login",
			},
		},
	)
	if err != nil {
		panic(err)
	}

	if m := re.FindStringSubmatch(resp.String()); m != nil {
		requestToken = m[1]
	}

	resp, err = session.Get("https://account.box.com/gen204?category=login&event_type=PASSWORD_AUTOFILLED_NO&keys_and_values%5BpageType%5D=twostage", nil)
	if err != nil {
		panic(err)
	}

	resp, err = session.Post(
		"https://account.box.com/login/credentials?redirect_url=%2F",
		&grequests.RequestOptions{
			Data: map[string]string{
				"login":         "@ed.tus.ac.jp",
				"password":      "",
				"_pw_sql":       "",
				"request_token": requestToken,
				"redirect_url":  "/",
			},
			Headers: map[string]string{
				"Sec-Fetch-Dest":            "document",
				"Sec-Fetch-Mode":            "navigate",
				"Sec-Fetch-Site":            "same-origin",
				"Sec-Fetch-User":            "?1",
				"Upgrade-Insecure-Requests": "1",
				"Referer":                   "https://account.box.com/login?redirect_url=/",
			},
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Header)
	// fmt.Println(resp.String())

	// req, err = http.NewRequest("GET", resp.Header["Location"][0], strings.NewReader(args.Encode()))
	// req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36 Edg/86.0.622.51")
	// req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// req.Header.Add("Connection", "keep-alive")
	// req.Header.Add("Accept", "*/*")
	// for _, c := range resp.Cookies() {
	// 	cookies = append(cookies, c)
	// }
	// for _, c := range cookies {
	// 	req.AddCookie(c)
	// }

	// resp, err = client.Do(req)
	// if err != nil {
	// 	panic(err)
	// }
	// defer resp.Body.Close()
}

// GetContent -
func (c *Client) GetContent(URL string) (string, error) {
	resp, err := c.session.Get(URL, nil)
	if err != nil {
		return "", err
	}

	return resp.String(), nil
}

// GetFileID get file id
func (c *Client) GetFileID(content string) (string, error) {
	re := regexp.MustCompile(`"\\/app-api\\/enduserapp\\/item\\/f_(\d+)"`)
	if fileID := re.FindStringSubmatch(content); fileID != nil {
		return fileID[1], nil
	}
	return "", nil
}

// GetRequestToken request token
func (c *Client) GetRequestToken(content string) (string, error) {
	re := regexp.MustCompile(`"requestToken":"(.+?)"`)
	if matchs := re.FindStringSubmatch(content); matchs != nil {
		return matchs[1], nil
	}
	return "", nil
}

// GetTokens write read token
func (c *Client) GetTokens(fileID string, requestToken string, sharedName string) (*Tokens, error) {
	resp, err := c.session.Post(
		"https://tus.app.box.com/app-api/enduserapp/elements/tokens",
		&grequests.RequestOptions{
			JSON: map[string]interface{}{
				"fileIDs": []string{"file_" + fileID},
			},
			Headers: map[string]string{
				"Request-Token":        requestToken,
				"X-Box-Client-Name":    "enduserapp",
				"X-Box-Client-Version": "20.364.1",
				"X-Box-EndUser-API":    "sharedName=" + sharedName,
				"X-Request-Token":      requestToken,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	jsonBody := map[string]Tokens{}
	err = resp.JSON(&jsonBody)
	if err != nil {
		return nil, err
	}

	// return fisrt (original one)
	for _, tokens := range jsonBody {
		return &tokens, nil
	}

	return nil, nil
}

// GetInfo file info
func (c *Client) GetInfo(writeToken string, fileID string, sharedName string) (*Info, error) {
	resp, err := c.session.Get(
		"https://api.box.com/2.0/files/"+fileID+"?fields=permissions,shared_link,sha1,file_version,name,size,extension,representations,watermark_info,authenticated_download_url,is_download_available",
		&grequests.RequestOptions{
			JSON: map[string]interface{}{
				"fileIDs": []string{"file_" + fileID},
			},
			Headers: map[string]string{
				"Authorization":     "Bearer " + writeToken,
				"X-Box-Client-Name": "ContentPreview",
				"BoxApi":            "shared_link=https://tus.app.box.com/s/" + sharedName,
				"X-Rep-Hints":       "[3d][pdf][text][mp3][json][jpg?dimensions=1024x1024&paged=false][jpg?dimensions=2048x2048,png?dimensions=2048x2048][dash,mp4][filmstrip]",
			},
		},
	)
	if err != nil {
		return nil, err
	}

	info := Info{}
	err = resp.JSON(&info)
	return &info, nil
}

// GetManifest get manifest
func (c *Client) GetManifest(readToken string, versionID string, fileID string, sharedName string) (string, error) {
	resp, err := c.session.Get(
		"https://dl.boxcloud.com/api/2.0/internal_files/"+fileID+
			"/versions/"+versionID+
			"/representations/dash/content/manifest.mpd?access_token="+readToken+
			"&shared_link=https%3A%2F%2Ftus.app.box.com%2Fs%2F"+sharedName+
			"&box_client_name=box-content-preview&box_client_version=2.52.0",
		nil,
	)
	if err != nil {
		return "", err
	}

	return resp.String(), nil
}

// DownloadFile download file
func (c *Client) DownloadFile(
	readToken string,
	versionID string,
	filename string,
	fileID string,
	sharedName string,
	resolution string,
	chunkNum int,
	threads int,
	docker bool,
) error {
	counter := &WriteCounter{}
	part := "init"

	var videoChunks []Chunk
	var audioChunks []Chunk

	for i := 0; i < chunkNum; i++ {
		waitGroutp.Add(1)
		go func(i int) {
			defer waitGroutp.Done()

			if i != 0 {
				part = strconv.Itoa(i)
			} else {
				part = "init"
			}

			vURL := "https://dl.boxcloud.com/api/2.0/internal_files/" + fileID +
				"/versions/" + versionID +
				"/representations/dash/content/video/" + resolution +
				"/" + part +
				".m4s?access_token=" + readToken +
				"&shared_link=https%3A%2F%2Ftus.app.box.com%2Fs%2F" + sharedName +
				"&box_client_name=box-content-preview&box_client_version=2.52.0"
			aURL := "https://dl.boxcloud.com/api/2.0/internal_files/" + fileID +
				"/versions/" + versionID +
				"/representations/dash/content/audio/0/" + part +
				".m4s?access_token=" + readToken +
				"&shared_link=https%3A%2F%2Ftus.app.box.com%2Fs%2F" + sharedName +
				"&box_client_name=box-content-preview&box_client_version=2.52.0"

			// video
			vdata, err := c.downloadPart(counter, vURL, readToken, sharedName)
			if err != nil {
				panic(err)
			}

			// audio
			adata, err := c.downloadPart(counter, aURL, readToken, sharedName)
			if err != nil {
				panic(err)
			}

			videoChunks = append(videoChunks, Chunk{Data: vdata, Index: i})
			audioChunks = append(audioChunks, Chunk{Data: adata, Index: i})
		}(i)

		if i%threads == 0 {
			waitGroutp.Wait()
		}
	}

	waitGroutp.Wait()

	fmt.Println()
	fmt.Println("Sort video and audio...")

	sort.SliceStable(videoChunks, func(i, j int) bool {
		return videoChunks[i].Index < videoChunks[j].Index
	})
	sort.SliceStable(audioChunks, func(i, j int) bool {
		return audioChunks[i].Index < audioChunks[j].Index
	})

	fmt.Println("Write video and audio...")

	p, err := os.Getwd()
	if err != nil {
		return err
	}

	tempPath := path.Join(p, "temp")
	_, err = os.Stat(tempPath)
	if os.IsNotExist(err) {
		err := os.Mkdir(tempPath, 0755)
		if err != nil {
			return err
		}
	}

	video := path.Join(tempPath, filename+".mp4")
	audio := path.Join(tempPath, filename+".mp3")

	vf, err := os.Create(video)
	if err != nil {
		return err
	}
	af, err := os.Create(audio)
	if err != nil {
		return err
	}

	for i := 0; i < len(videoChunks); i++ {
		if _, err = vf.Write(videoChunks[i].Data); err != nil {
			return err
		}
		if _, err = af.Write(audioChunks[i].Data); err != nil {
			return err
		}
	}

	vf.Close()
	af.Close()

	fmt.Println("Merge video and audio...")

	if docker {
		err = exec.Command(
			"docker", "run", "--rm", "-v", p+":/tmp", "jrottenberg/ffmpeg:4.0-scratch",
			"-y",
			"-i", path.Join("/tmp/temp", filename+".mp4"),
			"-i", path.Join("/tmp/temp", filename+".mp3"),
			"-c:v", "copy", "-c:a", "copy", "-strict", "experimental",
			path.Join("/tmp", filename),
		).Run()
	} else {
		err = exec.Command(
			"ffmpeg",
			"-y",
			"-i", video,
			"-i", audio,
			"-c:v", "copy", "-c:a", "copy", "-strict", "experimental",
			filename,
		).Run()
	}
	if err != nil {
		return err
	}

	fmt.Println("Merge video and audio finished !")

	if err = os.Remove(video); err != nil {
		return err
	}
	if err = os.Remove(audio); err != nil {
		return err
	}

	return nil
}

func (c *Client) downloadPart(
	counter *WriteCounter,
	url string,
	readToken string,
	sharedName string,
) ([]byte, error) {
	resp, err := c.session.Get(url, nil)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(io.TeeReader(resp.RawResponse.Body, counter))
}
