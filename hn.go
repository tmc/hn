// command hn provides a simple cli interface to hacker news
package main

import (
	"code.google.com/p/cascadia"
	"code.google.com/p/go.net/html"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/toqueteos/webbrowser"
	"github.com/wsxiaoys/terminal/color"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var (
	BaseURL = "https://news.ycombinator.com/"
)

type Entry struct {
	Id    int
	Title string
	Link  string
	Site  string
}

type Entries struct {
	Entries []Entry
	Next    string
}

// Parses the HTML in the source Reader to attempt to locate Entry objects
func ParseEntries(source io.Reader) (*Entries, error) {
	doc, err := html.Parse(source)
	if err != nil {
		return nil, err
	}

	sel := cascadia.MustCompile(`body center table tr td table tbody
								tr:not([style])`)

	entryFragments := sel.MatchAll(doc)

	result := make([]Entry, 0)
	// skip first
	for i := 1; i < len(entryFragments)-2; i += 2 {
		row1, row2 := entryFragments[i], entryFragments[i+1]
		entry, err := parseEntryFromFragments(row1, row2)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error parsing entry:", err)
		} else {
			result = append(result, entry)
		}
	}

	links := cascadia.MustCompile(`body table table td.title a`).MatchAll(doc)
	lastLink := links[len(links)-1]

	return &Entries{result, lastLink.Attr[0].Val}, nil
}

// Given two Nodes that make up the interesting tr elements of a post, populate
// an Entry
func parseEntryFromFragments(row1, row2 *html.Node) (Entry, error) {
	a := goquery.NewDocumentFromNode(row1)
	b := goquery.NewDocumentFromNode(row2)
	_ = b

	link := a.Find("td.title a")
	href, _ := link.Attr("href")

	idText := a.Find("td.title:first-child").Text()
	id, _ := strconv.Atoi(strings.TrimRight(idText, "."))

	e := Entry{
		Id:    id,
		Title: link.Text(),
		Site:  link.Parent().Find("span").Text(),
		Link:  href,
	}

	return e, nil
}

// Resolves multiple strings as url references
func combineUrls(urls ...string) string {
	result, _ := url.Parse("")
	for _, u := range urls {
		tmp, _ := url.Parse(u)
		result = result.ResolveReference(tmp)
	}
	return result.String()
}

// Fetches, parses and renders entries. the urlEndpoint parameter is appended to
// BaseURL to make up the url to fetch, prevEndpoint is used if the user selects
// the 'previous' page.
func fetchAndPrintEntries(urlEndpoint, prevEndpoint string) {
	url := combineUrls(BaseURL, urlEndpoint)
	fmt.Println("Fetching", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	entries, err := ParseEntries(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, entry := range entries.Entries {
		color.Printf("@y[%d] @{w!}%s @g%s\n", entry.Id, entry.Title, entry.Site)
		color.Println("   @w", entry.Link)
	}

	fmt.Println("Enter an article number to view it, n/p for next/previous")
	for {
		var input string
		fmt.Scan(&input)
		index, _ := strconv.Atoi(input)

		if input == "n" {
			fetchAndPrintEntries(entries.Next, urlEndpoint)
			break
		} else if input == "p" {
			fetchAndPrintEntries(prevEndpoint, urlEndpoint)
			break
		} else {
			for _, entry := range entries.Entries {
				if index == entry.Id {
					webbrowser.Open(entry.Link)
				}
			}
		}
	}
}

func main() {
	// use "news" as the default endpoint
	endpoint := "news"
	if len(os.Args) > 1 {
		endpoint = os.Args[1]
	}
	fetchAndPrintEntries(endpoint, "")
}
