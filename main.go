package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

// YouTube API 응답을 위한 구조체
type YouTubeResponse struct {
	Items []struct {
		Snippet struct {
			TopLevelComment struct {
				Snippet struct {
					AuthorDisplayName string `json:"authorDisplayName"`
					TextDisplay       string `json:"textDisplay"`
					AuthorChannelId   struct {
						Value string `json:"value"`
					} `json:"authorChannelId"`
				} `json:"snippet"`
			} `json:"topLevelComment"`
		} `json:"snippet"`
	} `json:"items"`
}

// URL에서 영상 ID 추출
func getVideoID(url string) string {
	if strings.Contains(url, "youtube.com/watch?v=") {
		splitURL := strings.Split(url, "v=")
		if len(splitURL) > 1 {
			return strings.Split(splitURL[1], "&")[0]
		}
	}
	return ""
}

func getVideoOwnerChannelId(apiKey string, videoId string) (string, error) {
	videoInfoUrl := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?part=snippet&id=%s&key=%s", videoId, apiKey)
	resp, err := http.Get(videoInfoUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var videoInfo struct {
		Items []struct {
			Snippet struct {
				ChannelId string `json:"channelId"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &videoInfo); err != nil {
		return "", err
	}

	if len(videoInfo.Items) > 0 {
		return videoInfo.Items[0].Snippet.ChannelId, nil
	}

	return "", fmt.Errorf("No video information found")
}

func main() {

	// .env 파일에서 환경 변수 로드
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	// 환경 변수에서 API 키 가져오기
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: No API key found in .env file")
		return
	}

	// 사용자로부터 유튜브 영상 URL 입력 받기
	var youtubeURL string
	fmt.Print("Enter YouTube video URL: ")
	fmt.Scanln(&youtubeURL)

	// URL에서 영상 ID 추출
	videoID := getVideoID(youtubeURL)

	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/commentThreads?key=%s&textFormat=plainText&part=snippet&videoId=%s", apiKey, videoID)

	// 영상 주인의 채널 ID 조회
	videoOwnerChannelId, err := getVideoOwnerChannelId(apiKey, videoID)
	if err != nil {
		fmt.Println("Error getting video owner channel ID:", err)
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("HTTP request failed:", err)
		return
	}
	defer resp.Body.Close()

	// 응답 상태 코드 확인
	fmt.Println("Status Code:", resp.StatusCode)

	// 응답 본문 읽기
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		return
	}

	// 응답 본문 로깅
	// fmt.Println("Response Body:", string(body))

	// JSON 응답 파싱

	var ytResponse YouTubeResponse
	if err := json.Unmarshal(body, &ytResponse); err != nil {
		fmt.Println("Failed to unmarshal JSON:", err)
		return
	}

	// 한글 댓글 필터링을 위한 정규 표현식
	re := regexp.MustCompile("[가-힣]")

	// 한글 댓글 파일 생성
	fileKorean, err := os.Create("comments_korean.csv")
	if err != nil {
		panic(err)
	}
	defer fileKorean.Close()

	writerKorean := csv.NewWriter(fileKorean)
	defer writerKorean.Flush()

	// 모든 댓글 파일 생성
	fileAll, err := os.Create("comments_all.csv")
	if err != nil {
		panic(err)
	}
	defer fileAll.Close()

	writerAll := csv.NewWriter(fileAll)
	defer writerAll.Flush()

	// CSV 파일 헤더 작성
	writerKorean.Write([]string{"Author", "Comment"})
	writerAll.Write([]string{"Author", "Comment"})

	for _, item := range ytResponse.Items {
		comment := item.Snippet.TopLevelComment.Snippet

		// 영상 주인의 댓글은 제외
		if comment.AuthorChannelId.Value == videoOwnerChannelId {
			continue
		}

		// 모든 댓글을 저장
		writerAll.Write([]string{comment.AuthorDisplayName, comment.TextDisplay})

		// 한글이 포함된 댓글만 필터링하여 저장
		if re.MatchString(comment.TextDisplay) {
			writerKorean.Write([]string{comment.AuthorDisplayName, comment.TextDisplay})
		}
	}
}
