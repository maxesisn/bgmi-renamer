package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
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
	var qBitBaseURL, qBitUsername, qBitPassword, openaiToken, openaiBaseURL, torrentCategory, torrentPath, torrentHashV1 string

	flag.StringVar(&qBitBaseURL, "qbit-url", "", "qBittorrent base URL")
	flag.StringVar(&qBitUsername, "qbit-username", "", "qBittorrent username")
	flag.StringVar(&qBitPassword, "qbit-password", "", "qBittorrent password")
	flag.StringVar(&openaiToken, "openai-token", "", "Your OpenAI token")
	flag.StringVar(&openaiBaseURL, "openai-url", "https://api.openai.com", "OpenAI API base URL")
	flag.StringVar(&torrentCategory, "category", "BGmi", "Torrent category")
	flag.StringVar(&torrentPath, "path", "", "Torrent path")
	flag.StringVar(&torrentHashV1, "hash", "", "Torrent hash (v1)")

	flag.Parse()

	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exPath := filepath.Dir(ex)

	configFile := filepath.Join(exPath, "bgmi-renamer.conf")

	if _, err := os.Stat(configFile); err == nil {
		file, err := os.Open(configFile)
		if err != nil {
			log.Printf("无法打开配置文件: %v\n", err)
		} else {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "qbit-url=") && qBitBaseURL == "" {
					qBitBaseURL = strings.TrimPrefix(line, "qbit-url=")
				} else if strings.HasPrefix(line, "qbit-username=") && qBitUsername == "" {
					qBitUsername = strings.TrimPrefix(line, "qbit-username=")
				} else if strings.HasPrefix(line, "qbit-password=") && qBitPassword == "" {
					qBitPassword = strings.TrimPrefix(line, "qbit-password=")
				} else if strings.HasPrefix(line, "openai-token=") && openaiToken == "" {
					openaiToken = strings.TrimPrefix(line, "openai-token=")
				} else if strings.HasPrefix(line, "openai-url=") && openaiBaseURL == "https://api.openai.com" {
					openaiBaseURL = strings.TrimPrefix(line, "openai-url=")
				}
			}
			if err := scanner.Err(); err != nil {
				log.Printf("读取配置文件出错: %v\n", err)
			}
		}
	}

	if qBitBaseURL == "" || qBitUsername == "" || qBitPassword == "" || openaiToken == "" ||
		torrentCategory == "" || torrentPath == "" || torrentHashV1 == "" {
		flag.Usage()
		log.Fatalf("Missing required arguments\n")
		return
	}

	if torrentCategory != "BGmi" {
		fmt.Println("Only category BGmi is supported")
		return
	}

	logFile := filepath.Join(exPath, "bgmi-renamer.log")
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Logging to", logFile)
	defer f.Close()
	log.SetOutput(f)
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("user.Current error: %v\n", err)
		return
	}
	log.Printf("bgmi-renamer started -> %s [user:%s]\n", torrentPath, currentUser.Username)

	openAIConfig := openai.DefaultConfig(openaiToken)
	openAIConfig.BaseURL = openaiBaseURL
	client := openai.NewClientWithConfig(openAIConfig)
	torrentName := filepath.Base(torrentPath)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
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
		log.Printf("Removing empty directory %s\n", torrentParentDir)
		// remove owner check so windows build can pass

		// if runtime.GOOS != "windows" {
		// 	// get owner of the directory, check if it's the same as the current user
		// 	dirStat, err := os.Stat(torrentParentDir)
		// 	if err != nil {
		// 		log.Fatalf("Stat error: %v\n", err)
		// 		return
		// 	}
		// 	dirOwner, err := user.LookupId(fmt.Sprintf("%d", dirStat.Sys().(*syscall.Stat_t).Uid))
		// 	if err != nil {
		// 		log.Fatalf("LookupId error: %v\n", err)
		// 		return
		// 	}
		// 	if dirOwner.Username != currentUser.Username {
		// 		log.Println("WARNING: Directory owner is not the same as the current user")
		// 	}
		// }
		err = os.Remove(torrentParentDir)
		if err != nil {
			log.Fatalf("Remove error: %v\n", err)
			return
		}
	} else {
		log.Printf("Directory %s is not empty\n", torrentParentDir)
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
