package scraper

import (
	"testing"
)

func TestSearchContentWithType(t *testing.T) {
	s := New("", nil)

	movies, err := s.SearchContentWithType("2026", "movie")
	if err != nil {
		t.Logf("SearchContentWithType movie err: %v", err)
	} else {
		for _, m := range movies {
			if m.MediaType != "Movie" {
				t.Errorf("Expected movie, got %s for title %s", m.MediaType, m.Title)
			}
		}
	}

	series, err := s.SearchContentWithType("2026", "tv_series")
	if err != nil {
		t.Logf("SearchContentWithType series err: %v", err)
	} else {
		for _, s := range series {
			if s.MediaType != "TV Series" {
				t.Errorf("Expected TV Series, got %s for title %s", s.MediaType, s.Title)
			}
		}
	}
}
