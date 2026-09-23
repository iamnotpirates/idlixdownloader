package extractor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iamnotpirates/idlixdownloader/internal/httpclient"
	"github.com/iamnotpirates/idlixdownloader/internal/models"
)

type IdlixClient struct {
	BaseURL string
	client  *httpclient.CurlClient
}

func New(baseURL string, client *httpclient.CurlClient) *IdlixClient {
	if baseURL == "" {
		baseURL = "https://z2.idlixku.com"
	}
	if client == nil {
		client = httpclient.New()
	}
	return &IdlixClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (c *IdlixClient) FetchMovieDetails(slug string) (*models.MovieDetails, error) {
	parts := strings.Split(strings.Trim(slug, "/"), "/")
	cleanSlug := parts[len(parts)-1]
	if cleanSlug == "" {
		cleanSlug = slug
	}

	endpoint := fmt.Sprintf("%s/api/movies/%s", c.BaseURL, cleanSlug)
	referer := fmt.Sprintf("%s/movie/%s", c.BaseURL, cleanSlug)

	body, err := c.client.Get(endpoint, referer, true)
	if err != nil {
		return nil, fmt.Errorf("movie metadata request failed: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, fmt.Errorf("failed to parse movie JSON: %w", err)
	}

	id := ""
	if v, ok := data["id"].(string); ok {
		id = v
	} else if v, ok := data["id"].(float64); ok {
		id = fmt.Sprintf("%.0f", v)
	}

	title := cleanSlug
	if v, ok := data["title"].(string); ok && v != "" {
		title = v
	}

	year := "N/A"
	if v, ok := data["releaseDate"].(string); ok && len(v) >= 4 {
		year = v[:4]
	} else if v, ok := data["year"].(string); ok && len(v) >= 4 {
		year = v[:4]
	}

	var synopsis *string
	if v, ok := data["overview"].(string); ok && v != "" {
		synopsis = &v
	}

	var poster *string
	if v, ok := data["posterPath"].(string); ok && v != "" {
		if strings.HasPrefix(v, "http") {
			poster = &v
		} else {
			p := fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", v)
			poster = &p
		}
	} else if v, ok := data["poster"].(string); ok && v != "" {
		if strings.HasPrefix(v, "http") {
			poster = &v
		} else {
			p := fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", v)
			poster = &p
		}
	}

	var runtime *int
	if v, ok := data["runtime"].(float64); ok && v > 0 {
		r := int(v)
		runtime = &r
	}

	var quality *string
	if v, ok := data["quality"].(string); ok && v != "" {
		quality = &v
	}

	var genres []string
	if arr, ok := data["genres"].([]any); ok {
		for _, g := range arr {
			if m, ok := g.(map[string]any); ok {
				if n, ok := m["name"].(string); ok {
					genres = append(genres, n)
				}
			}
		}
	}

	return &models.MovieDetails{
		ID:       id,
		Title:    title,
		Slug:     cleanSlug,
		Year:     year,
		Synopsis: synopsis,
		Poster:   poster,
		Runtime:  runtime,
		Quality:  quality,
		Genres:   genres,
	}, nil
}

func (c *IdlixClient) FetchSeriesDetails(slug string) (*models.SeriesDetails, error) {
	parts := strings.Split(strings.Trim(slug, "/"), "/")
	cleanSlug := parts[len(parts)-1]
	if cleanSlug == "" {
		cleanSlug = slug
	}

	endpoint := fmt.Sprintf("%s/api/series/%s", c.BaseURL, cleanSlug)
	referer := fmt.Sprintf("%s/series/%s", c.BaseURL, cleanSlug)

	body, err := c.client.Get(endpoint, referer, true)
	if err != nil {
		return nil, fmt.Errorf("series metadata request failed: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, fmt.Errorf("failed to parse series JSON: %w", err)
	}

	title := cleanSlug
	if v, ok := data["title"].(string); ok && v != "" {
		title = v
	}

	year := "N/A"
	if v, ok := data["firstAirDate"].(string); ok && len(v) >= 4 {
		year = v[:4]
	} else if v, ok := data["year"].(string); ok && len(v) >= 4 {
		year = v[:4]
	}

	var synopsis *string
	if v, ok := data["overview"].(string); ok && v != "" {
		synopsis = &v
	}

	var poster *string
	if v, ok := data["posterPath"].(string); ok && v != "" {
		if strings.HasPrefix(v, "http") {
			poster = &v
		} else {
			p := fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", v)
			poster = &p
		}
	}

	var seasons []models.SeasonInfo
	if sArr, ok := data["seasons"].([]any); ok {
		for _, sVal := range sArr {
			sMap, ok := sVal.(map[string]any)
			if !ok {
				continue
			}

			sNum := 1
			if v, ok := sMap["seasonNumber"].(float64); ok {
				sNum = int(v)
			} else if v, ok := sMap["season"].(float64); ok {
				sNum = int(v)
			}

			var episodes []models.EpisodeInfo
			if epsArr, ok := sMap["episodes"].([]any); ok && len(epsArr) > 0 {
				for _, epVal := range epsArr {
					epMap, ok := epVal.(map[string]any)
					if !ok {
						continue
					}
					epNum := 1
					if v, ok := epMap["episodeNumber"].(float64); ok {
						epNum = int(v)
					} else if v, ok := epMap["episode"].(float64); ok {
						epNum = int(v)
					}

					epTitle := fmt.Sprintf("Episode %d", epNum)
					if v, ok := epMap["name"].(string); ok && v != "" {
						epTitle = v
					} else if v, ok := epMap["title"].(string); ok && v != "" {
						epTitle = v
					}

					mediaID := ""
					if v, ok := epMap["id"].(string); ok {
						mediaID = v
					} else if v, ok := epMap["id"].(float64); ok {
						mediaID = fmt.Sprintf("%.0f", v)
					}

					epSlug := ""
					if v, ok := epMap["slug"].(string); ok {
						epSlug = v
					}

					episodes = append(episodes, models.EpisodeInfo{
						SeasonNum:  sNum,
						EpisodeNum: epNum,
						Title:      epTitle,
						MediaID:    mediaID,
						Slug:       epSlug,
					})
				}
			} else {
				// Fetch season episodes explicitly
				sEndpoint := fmt.Sprintf("%s/api/series/%s/season/%d", c.BaseURL, cleanSlug, sNum)
				if sBody, err := c.client.Get(sEndpoint, referer, true); err == nil {
					var sData map[string]any
					if err := json.Unmarshal([]byte(sBody), &sData); err == nil {
						if seasonObj, ok := sData["season"].(map[string]any); ok {
							if epsArr, ok := seasonObj["episodes"].([]any); ok {
								for _, epVal := range epsArr {
									epMap, ok := epVal.(map[string]any)
									if !ok {
										continue
									}
									epNum := 1
									if v, ok := epMap["episodeNumber"].(float64); ok {
										epNum = int(v)
									} else if v, ok := epMap["episode"].(float64); ok {
										epNum = int(v)
									}

									epTitle := fmt.Sprintf("Episode %d", epNum)
									if v, ok := epMap["name"].(string); ok && v != "" {
										epTitle = v
									}
									mediaID := ""
									if v, ok := epMap["id"].(string); ok {
										mediaID = v
									} else if v, ok := epMap["id"].(float64); ok {
										mediaID = fmt.Sprintf("%.0f", v)
									}
									epSlug := ""
									if v, ok := epMap["slug"].(string); ok {
										epSlug = v
									}

									episodes = append(episodes, models.EpisodeInfo{
										SeasonNum:  sNum,
										EpisodeNum: epNum,
										Title:      epTitle,
										MediaID:    mediaID,
										Slug:       epSlug,
									})
								}
							}
						}
					}
				}
			}

			seasons = append(seasons, models.SeasonInfo{
				SeasonNum: sNum,
				Episodes:  episodes,
			})
		}
	}

	return &models.SeriesDetails{
		Title:    title,
		Slug:     cleanSlug,
		Year:     year,
		Synopsis: synopsis,
		Poster:   poster,
		Seasons:  seasons,
	}, nil
}

func (c *IdlixClient) ExtractStream(contentType, slugOrID, pageURL string) (*models.StreamSources, error) {
	mediaID := slugOrID
	title := slugOrID

	if contentType == "movie" {
		parts := strings.Split(strings.Trim(pageURL, "/"), "/")
		cleanSlug := parts[len(parts)-1]
		if cleanSlug == "" {
			parts = strings.Split(strings.Trim(slugOrID, "/"), "/")
			cleanSlug = parts[len(parts)-1]
		}

		metaEndpoint := fmt.Sprintf("%s/api/movies/%s", c.BaseURL, cleanSlug)
		body, err := c.client.Get(metaEndpoint, pageURL, true)
		if err != nil {
			return nil, fmt.Errorf("failed to get movie metadata: %w", err)
		}

		var meta map[string]any
		if err := json.Unmarshal([]byte(body), &meta); err != nil {
			return nil, fmt.Errorf("invalid metadata JSON: %w", err)
		}

		if v, ok := meta["id"].(string); ok {
			mediaID = v
		} else if v, ok := meta["id"].(float64); ok {
			mediaID = fmt.Sprintf("%.0f", v)
		} else {
			return nil, fmt.Errorf("media ID not found in movie metadata")
		}

		if v, ok := meta["title"].(string); ok && v != "" {
			title = v
		} else {
			title = cleanSlug
		}
	}

	// 0. Ensure cookies exist
	if pageURL != "" {
		_, _ = c.client.Get(pageURL, "", false)
	} else {
		_, _ = c.client.Get(c.BaseURL, "", false)
	}

	// 1. Play-info gate token
	infoContentType := "movie"
	if contentType != "movie" {
		infoContentType = "episode"
	}

	playInfoURL := fmt.Sprintf("%s/api/watch/play-info/%s/%s", c.BaseURL, infoContentType, mediaID)
	body, err := c.client.Get(playInfoURL, pageURL, true)
	if err != nil {
		return nil, fmt.Errorf("failed to request play-info: %w", err)
	}

	var infoData map[string]any
	if err := json.Unmarshal([]byte(body), &infoData); err != nil {
		return nil, fmt.Errorf("invalid play-info JSON: %w", err)
	}

	gateToken, _ := infoData["gateToken"].(string)
	if gateToken == "" {
		return nil, fmt.Errorf("missing gateToken in response")
	}

	unlockAt, _ := infoData["unlockAt"].(float64)
	serverNow, _ := infoData["serverNow"].(float64)
	diff := (unlockAt - serverNow) / 1000.0
	waitSec := 0.5
	if diff > 0 {
		waitSec = diff + 1.0
		if waitSec > 30.0 {
			waitSec = 30.0
		}
	}

	if waitSec > 0 {
		fmt.Printf("[Extractor] Waiting %.1fs for gate countdown...\n", waitSec)
		time.Sleep(time.Duration(waitSec * float64(time.Second)))
	}

	// 2. Claim session
	claimURL := fmt.Sprintf("%s/api/watch/session/claim", c.BaseURL)
	claimPayload := map[string]string{"gateToken": gateToken}
	claimBody, err := c.client.PostJSON(claimURL, pageURL, claimPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to claim session: %w", err)
	}

	var claimData map[string]any
	if err := json.Unmarshal([]byte(claimBody), &claimData); err != nil {
		return nil, fmt.Errorf("invalid claim JSON: %w", err)
	}

	if errMsg, ok := claimData["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("session claim rejected: %s", errMsg)
	}

	claimToken, _ := claimData["claim"].(string)
	redeemURL, _ := claimData["redeemUrl"].(string)
	if claimToken == "" || redeemURL == "" {
		return nil, fmt.Errorf("missing claim token or redeemUrl")
	}

	// 3. Redeem master playlist URL and subtitles
	redeemPayload := map[string]string{"claim": claimToken}
	redeemBody, err := c.client.PostJSON(redeemURL, pageURL, redeemPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to redeem master playlist: %w", err)
	}

	var finalData map[string]any
	if err := json.Unmarshal([]byte(redeemBody), &finalData); err != nil {
		return nil, fmt.Errorf("invalid redeem JSON: %w", err)
	}

	if errMsg, ok := finalData["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("master playlist redeem rejected: %s", errMsg)
	}

	m3u8URL, _ := finalData["url"].(string)
	if m3u8URL == "" {
		return nil, fmt.Errorf("no master m3u8 playlist found in stream response")
	}

	var subtitles []models.SubtitleTrack
	if subsArr, ok := finalData["subtitles"].([]any); ok {
		for _, sVal := range subsArr {
			if sMap, ok := sVal.(map[string]any); ok {
				lang := "Subtitle"
				if l, ok := sMap["label"].(string); ok && l != "" {
					lang = l
				} else if l, ok := sMap["lang"].(string); ok && l != "" {
					lang = l
				}
				subURL, _ := sMap["path"].(string)
				if subURL != "" {
					subtitles = append(subtitles, models.SubtitleTrack{
						Lang: lang,
						URL:  subURL,
					})
				}
			}
		}
	}

	return &models.StreamSources{
		Title:     title,
		M3U8URL:   m3u8URL,
		Subtitles: subtitles,
	}, nil
}
