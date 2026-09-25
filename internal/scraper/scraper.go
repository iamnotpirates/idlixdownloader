package scraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/iamnotpirates/idlixdownloader/internal/httpclient"
	"github.com/iamnotpirates/idlixdownloader/internal/models"
)

type Scraper struct {
	BaseURL string
	client  *httpclient.CurlClient
}

func New(baseURL string, client *httpclient.CurlClient) *Scraper {
	if baseURL == "" {
		baseURL = "https://z2.idlixku.com"
	}
	if client == nil {
		client = httpclient.New()
	}
	return &Scraper{
		BaseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (s *Scraper) SearchContent(query string) ([]models.MediaItem, error) {
	return s.SearchContentWithType(query, "")
}

func (s *Scraper) SearchContentWithType(query, typeFilter string) ([]models.MediaItem, error) {
	qClean := strings.TrimSpace(query)
	if qClean == "" {
		return nil, nil
	}

	apiURL := fmt.Sprintf("%s/api/search?q=%s", s.BaseURL, url.QueryEscape(qClean))
	if typeFilter != "" {
		validType := typeFilter
		if typeFilter == "series" || typeFilter == "tv" {
			validType = "tv_series"
		}
		apiURL = fmt.Sprintf("%s/api/search?q=%s&type=%s", s.BaseURL, url.QueryEscape(qClean), url.QueryEscape(validType))
	}
	referer := fmt.Sprintf("%s/", s.BaseURL)

	body, err := s.client.Get(apiURL, referer, true)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	var data struct {
		Results []struct {
			Title        string   `json:"title"`
			Name         string   `json:"name"`
			Slug         string   `json:"slug"`
			ContentType  string   `json:"contentType"`
			ReleaseDate  string   `json:"releaseDate"`
			FirstAirDate string   `json:"firstAirDate"`
			VoteAverage  *float64 `json:"voteAverage"`
			PosterPath   string   `json:"posterPath"`
			Poster       string   `json:"poster"`
		} `json:"results"`
	}

	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, fmt.Errorf("failed to parse search JSON: %w", err)
	}

	var items []models.MediaItem
	for _, item := range data.Results {
		title := item.Title
		if title == "" {
			title = item.Name
		}
		if title == "" {
			title = "Unknown"
		}

		isTV := item.ContentType == "tv_series" ||
			item.ContentType == "series" ||
			item.ContentType == "tvshows" ||
			strings.Contains(item.Slug, "series/") ||
			strings.Contains(item.Slug, "tvshows/")

		if typeFilter == "movie" && isTV {
			continue
		}
		if (typeFilter == "tv_series" || typeFilter == "series" || typeFilter == "tv") && !isTV {
			continue
		}

		mediaType := "Movie"
		itemURL := fmt.Sprintf("%s/movie/%s", s.BaseURL, item.Slug)
		if isTV {
			mediaType = "TV Series"
			itemURL = fmt.Sprintf("%s/series/%s", s.BaseURL, item.Slug)
		}

		relDate := item.ReleaseDate
		if relDate == "" {
			relDate = item.FirstAirDate
		}
		var year *string
		if len(relDate) >= 4 {
			y := relDate[:4]
			year = &y
		}

		rating := "N/A"
		if item.VoteAverage != nil {
			rating = fmt.Sprintf("%.1f", *item.VoteAverage)
		}

		rawPoster := item.PosterPath
		if rawPoster == "" {
			rawPoster = item.Poster
		}
		poster := ""
		if rawPoster != "" {
			if strings.HasPrefix(rawPoster, "http") {
				poster = rawPoster
			} else {
				poster = fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", rawPoster)
			}
		}

		items = append(items, models.MediaItem{
			Title:     title,
			URL:       itemURL,
			Slug:      item.Slug,
			Rating:    rating,
			MediaType: mediaType,
			Poster:    poster,
			Year:      year,
		})
	}

	return items, nil
}

func (s *Scraper) FetchFeatured() ([]models.MediaItem, error) {
	html, err := s.client.Get(s.BaseURL, "", false)
	if err != nil {
		// Fallback to default search
		return s.SearchContent("2024")
	}

	type target struct {
		name string
		url  string
		slug string
		isTV bool
	}

	var targets []target

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err == nil {
		doc.Find("script[type='application/ld+json']").Each(func(i int, sel *goquery.Selection) {
			text := sel.Text()
			var jsonVals []map[string]any
			if err := json.Unmarshal([]byte(text), &jsonVals); err == nil {
				for _, obj := range jsonVals {
					if obj["@type"] == "ItemList" {
						if elems, ok := obj["itemListElement"].([]any); ok {
							for _, el := range elems {
								if m, ok := el.(map[string]any); ok {
									name, _ := m["name"].(string)
									u, _ := m["url"].(string)
									if name != "" && u != "" {
										parts := strings.Split(strings.Trim(u, "/"), "/")
										slug := parts[len(parts)-1]
										isTV := strings.Contains(u, "/series/")
										targets = append(targets, target{name: name, url: u, slug: slug, isTV: isTV})
									}
								}
							}
						}
					}
				}
			}
		})
	}

	if len(targets) == 0 {
		return s.SearchContent("2024")
	}

	var wg sync.WaitGroup
	items := make([]models.MediaItem, len(targets))

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, tgt target) {
			defer wg.Done()
			res, err := s.SearchContent(tgt.name)
			if err == nil {
				for _, r := range res {
					if r.Slug == tgt.slug || strings.EqualFold(r.Title, tgt.name) {
						items[idx] = r
						return
					}
				}
			}
			mType := "Movie"
			if tgt.isTV {
				mType = "TV Series"
			}
			items[idx] = models.MediaItem{
				Title:     tgt.name,
				URL:       tgt.url,
				Slug:      tgt.slug,
				Rating:    "Featured",
				MediaType: mType,
			}
		}(i, t)
	}
	wg.Wait()

	var result []models.MediaItem
	for _, it := range items {
		if it.Title != "" {
			result = append(result, it)
		}
	}

	return result, nil
}
