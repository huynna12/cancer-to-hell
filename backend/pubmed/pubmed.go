package pubmed

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	searchURL = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi"
	fetchURL  = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/efetch.fcgi"
)

// Paper is one PubMed article returned to the caller.
// Abstract holds the full abstract text so the model can reason from the actual content,
// not just cite from the title/metadata.
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
// PubMed wraps gene names (e.g. <i>BRCA1/2</i>) inside ArticleTitle, so
// Go's default string decoder silently drops that text.
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
// Structured abstracts have multiple sections with a Label attribute
// (BACKGROUND, METHODS, RESULTS, CONCLUSIONS).
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
	ids, err := searchIDs(query)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []Paper{}, nil
	}
	return fetchPapers(ids)
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

	resp, err := http.Get(searchURL + "?" + params.Encode())
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

	resp, err := http.Get(fetchURL + "?" + params.Encode())
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

		// Build author list
		authors := make([]string, 0)
		for _, auth := range a.MedlineCitation.Article.AuthorList.Authors {
			if auth.LastName != "" {
				authors = append(authors, auth.LastName+", "+auth.Initials+".")
			}
		}

		// Build abstract — join structured sections with their labels
		// e.g. "BACKGROUND: ... METHODS: ... RESULTS: ... CONCLUSIONS: ..."
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
