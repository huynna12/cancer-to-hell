package pubmed

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	searchURL = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi"
	fetchURL  = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/efetch.fcgi"
)

// Paper represents a single PubMed research paper
type Paper struct {
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	Journal string   `json:"journal"`
	Year    string   `json:"year"`
	PMID    string   `json:"pmid"`
	Link    string   `json:"link"`
}

// searchResult parses the esearch XML response
type searchResult struct {
	IDs []string `xml:"IdList>Id"`
}

// fetchResult parses the efetch XML response
type fetchResult struct {
	Articles []article `xml:"PubmedArticle"`
}

type article struct {
	MedlineCitation medlineCitation `xml:"MedlineCitation"`
}

type medlineCitation struct {
	PMID       string     `xml:"PMID"`
	Article    article2   `xml:"Article"`
	AuthorList authorList `xml:"Article>AuthorList"`
}

type authorList struct {
	Authors []author `xml:"Author"`
}

type author struct {
	LastName string `xml:"LastName"`
	Initials string `xml:"Initials"`
}

type article2 struct {
	Title   string  `xml:"ArticleTitle"`
	Journal journal `xml:"Journal"`
}

type journal struct {
	Title string `xml:"Title"`
	Year  string `xml:"JournalIssue>PubDate>Year"`
}

// Search queries PubMed and returns top 5 relevant papers
func Search(query string) ([]Paper, error) {
	ids, err := searchIDs(query)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []Paper{}, nil
	}

	papers, err := fetchPapers(ids)
	if err != nil {
		return nil, err
	}

	return papers, nil
}

func searchIDs(query string) ([]string, error) {
	params := url.Values{}
	params.Set("db", "pubmed")
	params.Set("term", query)
	params.Set("retmax", "5")
	params.Set("sort", "date")
	params.Set("retmode", "xml")

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
		for _, auth := range a.MedlineCitation.AuthorList.Authors {
			if auth.LastName != "" {
				authors = append(authors, auth.LastName+", "+auth.Initials+".")
			}
		}

		papers = append(papers, Paper{
			Title:   a.MedlineCitation.Article.Title,
			Authors: authors,
			Journal: a.MedlineCitation.Article.Journal.Title,
			Year:    a.MedlineCitation.Article.Journal.Year,
			PMID:    pmid,
			Link:    "https://pubmed.ncbi.nlm.nih.gov/" + pmid,
		})
	}

	return papers, nil
}
