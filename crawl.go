//Package crawl is A web crawler which crawls a given url to find similar domain urls
package crawl

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func get(url string) (*http.Response, error) {
	httpclient := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	res, err := httpclient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func parseLinks(page *goquery.Document) []string {
	parsedURLs := []string{}
	if page != nil {
		page.Find("a").Each(func(arg1 int, arg2 *goquery.Selection) {
			res, _ := arg2.Attr("href")
			parsedURLs = append(parsedURLs, res)
		})
		return parsedURLs
	}
	return parsedURLs
}

func getBaseURLs(baseURL string, urls []string) []string {
	baseurls := []string{}
	for _, surl := range urls {
		if strings.HasPrefix(surl, baseURL) {
			baseurls = append(baseurls, surl)
		} else if strings.HasPrefix("/", baseURL) {
			completedURL := baseURL + surl
			baseurls = append(baseurls, completedURL)
		}
	}
	return baseurls
}

func crawlPage(baseURL, targetURL string, mutx *sync.Mutex) []string {
	mutx.Lock()
	result, err := get(targetURL)
	mutx.Unlock()
	if err != nil {
		return nil
	}
	pageDoc, _ := goquery.NewDocumentFromResponse(result)
	allURLs := parseLinks(pageDoc)
	baseURLs := getBaseURLs(baseURL, allURLs)
	return baseURLs
}

func getDomainName(sourceURL string) string {
	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		return ""
	}
	return parsedURL.Scheme + "://" + parsedURL.Host
}

//Crawl takes the URL to be crawled and returns a slice URL strings
//We crawl the URL to find all the URL that starts with similar domain name and add into our channel
//These channels are read for data and further crawling is done on slice of URL strings
func Crawl(sourceURL string) []string {
	crawlablelinks := make(chan []string)
	var n int
	n++
	m := sync.Mutex{}

	//Go channels by default block until the other side is ready
	//So we need to send it through a go routine
	go func() {
		crawlablelinks <- []string{sourceURL}
	}()

	//A map to check if a URL was already crawled
	links := make(map[string]bool)
	domainName := getDomainName(sourceURL)
	//Slice of URL i.e string which were crawled
	results := []string{}
	for ; n > 0; n-- {
		fmt.Println("Slices of URLs in queue to be crawlled : ", n)
		//Read the slice from the channel
		workinglinks := <-crawlablelinks
		for _, value := range workinglinks {
			_, isParsed := links[value]
			//Check if the URL was already crawled
			if !isParsed {
				links[value] = true
				n++
				go func(domainName, value string, m *sync.Mutex) {
					foundlinks := crawlPage(domainName, value, m)
					results = append(results, foundlinks...)
					if foundlinks != nil {
						crawlablelinks <- foundlinks
					}
				}(domainName, value, &m)
			}
		}
	}
	return results
}
