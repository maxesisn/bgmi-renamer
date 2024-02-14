package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

//go:embed prompt.txt
var prompt []byte

type LLMResult struct {
	Season  int `json:"season"`
	Episode int `json:"episode"`
}

func main() {

	if len(os.Args) != 8 {
		log.Fatalf("Usage: bgmi-renamer <qBit base URL> <qBit username> <qBit password> <your openAI token> %%L %%F %%I\n")
		log.Fatalf("Example: bgmi-renamer http://localhost:8080 admin adminadmin sk-your_openai_token \"%%L\" \"%%F\" \"%%I\"\n")
		return
	}

	qBitBaseURL := os.Args[1]
	qBitUsername := os.Args[2]
	qBitPassword := os.Args[3]
	openaiToken := os.Args[4]
	torrentCategory := os.Args[5]
	torrentPath := os.Args[6]
	torrentHashV1 := os.Args[7]

	if torrentCategory != "BGmi" {
		fmt.Println("Only category BGmi is supported")
		return
	}

	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exPath := filepath.Dir(ex)

	logFile := filepath.Join(exPath, "bgmi-renamer.log")
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Printf("bgmi-renamer started -> %s\n", torrentPath)

	client := openai.NewClient(openaiToken)
	torrentName := filepath.Base(torrentPath)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo0125,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: string(prompt),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "[XKsub][Mobile Suit Gundam - The Witch from Mercury S2][12][CHT_JAP][1080P][WEBrip][MP4].mp4",
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: `{"season": 2, "episode": 12}`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "[ANi] 我內心的糟糕念頭 第二季 - 16 [1080P][Baha][WEB-DL][AAC AVC][CHT].mp4",
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: `{"season": 2, "episode": 16}`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "[Haretahoo.sub][Fate_kaleid_liner_3rei!!][11][GB][1080P] V2/[Haretahoo.sub][Fate_kaleid_liner_3rei!!][11][GB][1080P].mp4",
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: `{"season": 3, "episode": 11}`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "[夜莺家族][樱桃小丸子第二期(Chibi Maruko-chan II)][1421]小丸子想招来福气！&小丸子想去掉毛球[2024.01.28][GB_JP][1080P][MP4].mp4",
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: `{"season": 2, "episode": 1421}`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: torrentName,
				},
			},
		},
	)

	if err != nil {
		log.Fatalf("ChatCompletion error: %v\n", err)
		return
	}

	fmt.Println(resp.Choices[0].Message.Content)
	var result LLMResult
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result)
	if err != nil {
		log.Fatalf("Unmarshal error: %v\n", err)
		return
	}
	torrentParentDir := filepath.Dir(torrentPath)
	torrentParParentDir := filepath.Dir(torrentParentDir)
	torrentParParentDirName := filepath.Base(torrentParParentDir)
	seasonDir := filepath.Join(torrentParParentDir, fmt.Sprintf("Season %d", result.Season))
	torrentNameExt := filepath.Ext(torrentName)

	newFileName := fmt.Sprintf("%s - S%02dE%02d%s", torrentParParentDirName, result.Season, result.Episode, torrentNameExt)

	log.Printf("Moving %s to %s\n", torrentPath, seasonDir+"/"+newFileName)

	SID := LoginQBittorrent(qBitBaseURL, qBitUsername, qBitPassword)
	if SID == "" {
		log.Fatalln("LoginQBittorrent failed")
		return
	}

	MoveTorrent(qBitBaseURL, SID, torrentHashV1, seasonDir)
	RenameFile(qBitBaseURL, SID, torrentHashV1, torrentName, newFileName)

	// remove torrent parent directory if it's empty
	// sleep 60 seconds to let qbittorrent move the files
	time.Sleep(60 * time.Second)

	files, err := os.ReadDir(torrentParentDir)
	if err != nil {
		log.Fatalf("ReadDir error: %v\n", err)
		return
	}
	if len(files) == 0 {
		log.Fatalf("Removing %s\n", torrentParentDir)
		err = os.Remove(torrentParentDir)
		if err != nil {
			log.Fatalf("Remove error: %v\n", err)
			return
		}
	}
}

func LoginQBittorrent(base_url string, username string, password string) string {
	// login to qbittorrent
	base_url = strings.TrimSuffix(base_url, "/")

	fullUrl := fmt.Sprintf("%s/api/v2/auth/login", base_url)
	resp, err := http.PostForm(fullUrl, url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		log.Fatalf("PostForm error: %v\n", err)
		return ""
	}
	defer resp.Body.Close()
	if err != nil {
		log.Fatalf("ReadAll error: %v\n", err)
		return ""
	}
	cookies := resp.Cookies()
	// read Set-Cookie: SID=xxxxx
	for _, cookie := range cookies {
		if cookie.Name == "SID" {
			return cookie.Value
		}
	}
	return ""
}

func MoveTorrent(base_url string, SID string, hash string, newLocation string) {
	/*
		Set torrent location

		Requires knowing the torrent hash. You can get it from torrent list.

		POST /api/v2/torrents/setLocation HTTP/1.1
		User-Agent: Fiddler
		Host: 127.0.0.1
		Cookie: SID=your_sid
		Content-Type: application/x-www-form-urlencoded
		Content-Length: length
	*/
	// move torrent
	base_url = strings.TrimSuffix(base_url, "/")
	fullUrl := fmt.Sprintf("%s/api/v2/torrents/setLocation", base_url)

	// request with cookie SID=xxxxx
	req, err := http.NewRequest("POST", fullUrl, strings.NewReader(fmt.Sprintf("hashes=%s&location=%s", hash, newLocation)))
	if err != nil {
		log.Fatalf("NewRequest error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SID=%s", SID))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Do error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	if err != nil {
		log.Fatalf("ReadAll error: %v\n", err)
		return
	}
}

func RenameFile(base_url string, SID string, hash string, old_path string, new_path string) {
	// rename file
	base_url = strings.TrimSuffix(base_url, "/")
	fullUrl := fmt.Sprintf("%s/api/v2/torrents/renameFile", base_url)

	data := url.Values{}
	data.Set("hash", hash)
	data.Set("oldPath", old_path)
	data.Set("newPath", new_path)

	// request with cookie SID=xxxxx
	req, err := http.NewRequest("POST", fullUrl, strings.NewReader(data.Encode()))
	if err != nil {
		log.Fatalf("NewRequest error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SID=%s", SID))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Do error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	if err != nil {
		log.Fatalf("ReadAll error: %v\n", err)
		return
	}
	fmt.Println(resp)
}
