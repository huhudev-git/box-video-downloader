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
	"regexp"
	"strconv"
)

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
	req.Header.Set("X-Box-Client-Version", "20.286.0")
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
	req, _ := http.NewRequest("GET", "https://dl.boxcloud.com/api/2.0/internal_files/"+fileID+"/versions/"+versionID+"/representations/dash/content/manifest.mpd?access_token="+readToken+"&shared_link=https%3A%2F%2Ftus.app.box.com%2Fs%2F"+sharedName+"&box_client_name=box-content-preview&box_client_version=2.40.0", nil)

	bodyBytes, err := c.getContent(client, req)
	if err != nil {
		return "", err
	}
	bodyString := string(bodyBytes)
	return bodyString, nil
}

// DownloadFile download file
func (c *Client) DownloadFile(readToken string, versionID string, filename string, fileID string, sharedName string) error {
	video := "./temp/" + filename + ".mp4"
	audio := "./temp/" + filename + ".mp3"

	vf, err := os.Create(video)
	if err != nil {
		return err
	}
	// TODO: file already close
	defer vf.Close()

	af, err := os.Create(audio)
	if err != nil {
		return err
	}
	// TODO: file already close
	defer af.Close()

	client := &http.Client{}
	counter := &WriteCounter{}
	part := "init"

	// TODO: find end part index
	for i := 0; i < 100000; i++ {
		if i != 0 {
			part = strconv.Itoa(i)
		}

		vURL := "https://dl.boxcloud.com/api/2.0/internal_files/" + fileID + "/versions/" + versionID + "/representations/dash/content/video/1080/" + part + ".m4s?access_token=" + readToken + "&shared_link=https%3A%2F%2Ftus.app.box.com%2Fs%2F" + sharedName + "&box_client_name=box-content-preview&box_client_version=2.40.0"
		aURL := "https://dl.boxcloud.com/api/2.0/internal_files/" + fileID + "/versions/" + versionID + "/representations/dash/content/audio/0/" + part + ".m4s?access_token=" + readToken + "&shared_link=https%3A%2F%2Ftus.app.box.com%2Fs%2F" + sharedName + "&box_client_name=box-content-preview&box_client_version=2.40.0"

		// video
		vok, err := c.downloadPart(client, vf, counter, vURL)
		if err != nil {
			return err
		}

		// audio
		aok, err := c.downloadPart(client, af, counter, aURL)
		if err != nil {
			return err
		}

		if vok || aok {
			break
		}
	}

	fmt.Println()
	fmt.Println("Merge video and audio...")

	err = exec.Command("ffmpeg", "-i", video, "-i", audio, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental", filename).Run()
	if err != nil {
		return err
	}

	fmt.Println("Merge video and audio finished")

	vf.Close()
	if err = os.Remove(video); err != nil {
		return err
	}

	af.Close()
	if err = os.Remove(audio); err != nil {
		return err
	}

	return nil
}

func (c *Client) downloadPart(client *http.Client, f *os.File, counter *WriteCounter, url string) (bool, error) {
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return true, nil
	}

	body, err := ioutil.ReadAll(io.TeeReader(resp.Body, counter))
	if _, err = f.Write(body); err != nil {
		return false, err
	}

	return false, nil
}
