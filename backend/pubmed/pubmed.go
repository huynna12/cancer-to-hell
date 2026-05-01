package pubmed

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	searchURL = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi"
	fetchURL  = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/efetch.fcgi"
)

// cache avoids hitting PubMed repeatedly for the same query within a process lifetime.
var cache sync.Map // map[string][]Paper
var httpClient = &http.Client{Timeout: pubmedTimeout()}

// Paper is one PubMed article returned to the caller.
type Paper struct {
	Title    string   `json:"title"`
	Authors  []string `json:"authors"`
	Journal  string   `json:"journal"`
	Year     string   `json:"year"`
	PMID     string   `json:"pmid"`
	Link     string   `json:"link"`
	Abstract string   `json:"abstract"`
}

// ── XML helpers ──────────────────────────────────────────────────────────────

// xmlText collects ALL character data inside an XML element, including text
// nested inside child elements like <i>, <b>, <sup>, etc.
type xmlText struct {
	Value string
}

func (x *xmlText) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var buf strings.Builder
	depth := 1
	for depth > 0 {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch v := tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			buf.Write(v)
		}
	}
	x.Value = strings.TrimSpace(buf.String())
	return nil
}

// abstractSection maps to one <AbstractText> element.
type abstractSection struct {
	Label string
	Text  string
}

func (a *abstractSection) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "Label" {
			a.Label = strings.ToUpper(attr.Value)
		}
	}
	var buf strings.Builder
	depth := 1
	for depth > 0 {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch v := tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			buf.Write(v)
		}
	}
	a.Text = strings.TrimSpace(buf.String())
	return nil
}

type articleAbstract struct {
	Sections []abstractSection `xml:"AbstractText"`
}

// ── XML struct tree ──────────────────────────────────────────────────────────

type searchResult struct {
	IDs []string `xml:"IdList>Id"`
}

type fetchResult struct {
	Articles []article `xml:"PubmedArticle"`
}

type article struct {
	MedlineCitation medlineCitation `xml:"MedlineCitation"`
}

type medlineCitation struct {
	PMID    string   `xml:"PMID"`
	Article article2 `xml:"Article"`
}

type article2 struct {
	Title      xmlText         `xml:"ArticleTitle"`
	Abstract   articleAbstract `xml:"Abstract"`
	Journal    journal         `xml:"Journal"`
	AuthorList authorList      `xml:"AuthorList"`
}

type authorList struct {
	Authors []author `xml:"Author"`
}

type author struct {
	LastName string `xml:"LastName"`
	Initials string `xml:"Initials"`
}

type journal struct {
	Title string `xml:"Title"`
	Year  string `xml:"JournalIssue>PubDate>Year"`
}

// ── Public API ───────────────────────────────────────────────────────────────

func apiKey() string {
	return os.Getenv("NCBI_API_KEY")
}

func Search(query string) ([]Paper, error) {
	if cached, ok := cache.Load(query); ok {
		return cached.([]Paper), nil
	}

	ids, err := searchIDs(query)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		cache.Store(query, []Paper{})
		return []Paper{}, nil
	}

	papers, err := fetchPapers(ids)
	if err != nil {
		return nil, err
	}

	cache.Store(query, papers)
	return papers, nil
}

func searchIDs(query string) ([]string, error) {
	params := url.Values{}
	params.Set("db", "pubmed")
	params.Set("term", query)
	params.Set("retmax", "5")
	params.Set("retmode", "xml")
	if key := apiKey(); key != "" {
		params.Set("api_key", key)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pubmed search failed: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result searchResult
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.IDs, nil
}

func fetchPapers(ids []string) ([]Paper, error) {
	params := url.Values{}
	params.Set("db", "pubmed")
	params.Set("id", strings.Join(ids, ","))
	params.Set("retmode", "xml")
	params.Set("rettype", "abstract")
	if key := apiKey(); key != "" {
		params.Set("api_key", key)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fetchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pubmed fetch failed: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result fetchResult
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	papers := make([]Paper, 0)
	for _, a := range result.Articles {
		pmid := a.MedlineCitation.PMID

		authors := make([]string, 0)
		for _, auth := range a.MedlineCitation.Article.AuthorList.Authors {
			if auth.LastName != "" {
				authors = append(authors, auth.LastName+", "+auth.Initials+".")
			}
		}

		var abstractParts []string
		for _, s := range a.MedlineCitation.Article.Abstract.Sections {
			if s.Text == "" {
				continue
			}
			if s.Label != "" {
				abstractParts = append(abstractParts, s.Label+": "+s.Text)
			} else {
				abstractParts = append(abstractParts, s.Text)
			}
		}
		abstractText := strings.Join(abstractParts, " ")

		papers = append(papers, Paper{
			Title:    a.MedlineCitation.Article.Title.Value,
			Authors:  authors,
			Journal:  a.MedlineCitation.Article.Journal.Title,
			Year:     a.MedlineCitation.Article.Journal.Year,
			PMID:     pmid,
			Link:     "https://pubmed.ncbi.nlm.nih.gov/" + pmid,
			Abstract: abstractText,
		})
	}

	return papers, nil
}

func pubmedTimeout() time.Duration {
	raw := os.Getenv("PUBMED_TIMEOUT_SECONDS")
	if raw == "" {
		return 20 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 20 * time.Second
	}
	return time.Duration(seconds) * time.Second
}
