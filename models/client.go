package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"sync"
)

var waitGroutp = sync.WaitGroup{}

// Client box client
type Client struct {
	cookie string
}

// NewClient creates a new client
func NewClient(cookie string) *Client {
	return &Client{
		cookie: cookie,
	}
}

// GetContent get html content
func (c *Client) GetContent(url string) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Cookie", c.cookie)

	return c.getContent(client, req)
}

// GetContent get html content
func (c *Client) getContent(client *http.Client, req *http.Request) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return []byte(""), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []byte(""), err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte(""), err
	}
	return bodyBytes, nil
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
	client := &http.Client{}
	jsonStr := []byte(`{"fileIDs":["file_` + fileID + `"]}`)
	req, _ := http.NewRequest("POST", "https://tus.app.box.com/app-api/enduserapp/elements/tokens", bytes.NewBuffer(jsonStr))
	req.Header.Set("Cookie", c.cookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Request-Token", requestToken)
	req.Header.Set("X-Box-Client-Name", "enduserapp")
	req.Header.Set("X-Box-Client-Version", "20.364.1")
	req.Header.Set("X-Box-EndUser-API", "sharedName="+sharedName)
	req.Header.Set("X-Request-Token", requestToken)

	bodyBytes, err := c.getContent(client, req)
	if err != nil {
		return nil, err
	}

	jsonBody := map[string]Tokens{}
	err = json.Unmarshal(bodyBytes, &jsonBody)
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
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://api.box.com/2.0/files/"+fileID+"?fields=file_version,name,authenticated_download_url,is_download_available", nil)
	req.Header.Set("Authorization", "Bearer "+writeToken)
	req.Header.Set("X-Box-Client-Name", "ContentPreview")
	req.Header.Set("BoxApi", "shared_link=https://tus.app.box.com/s/"+sharedName)
	req.Header.Set("X-Rep-Hints", "[3d][pdf][text][mp3][json][jpg?dimensions=1024x1024&paged=false][jpg?dimensions=2048x2048,png?dimensions=2048x2048][dash,mp4][filmstrip]")

	bodyBytes, err := c.getContent(client, req)
	if err != nil {
		return nil, err
	}

	var info Info
	err = json.Unmarshal(bodyBytes, &info)
	return &info, nil
}

// GetManifest get manifest
func (c *Client) GetManifest(readToken string, versionID string, fileID string, sharedName string) (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(
		"GET",
		"https://dl.boxcloud.com/api/2.0/internal_files/"+fileID+
			"/versions/"+versionID+
			"/representations/dash/content/manifest.mpd?access_token="+readToken+
			"&shared_link=https%3A%2F%2Ftus.app.box.com%2Fs%2F"+sharedName+
			"&box_client_name=box-content-preview&box_client_version=2.52.0",
		nil,
	)

	bodyBytes, err := c.getContent(client, req)
	if err != nil {
		return "", err
	}
	bodyString := string(bodyBytes)
	return bodyString, nil
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

	client := &http.Client{}
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
			vdata, err := c.downloadPart(client, counter, vURL)
			if err != nil {
				panic(err)
			}

			// audio
			adata, err := c.downloadPart(client, counter, aURL)
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

func (c *Client) downloadPart(client *http.Client, counter *WriteCounter, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	return ioutil.ReadAll(io.TeeReader(resp.Body, counter))
}
