package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/gocolly/colly"
)

func StartScraping(logPath string) {
	searchTerms := extractSearchTerms(logPath)
	if len(searchTerms) == 0 {
		fmt.Println("No search terms found.")
		return
	}

	for _, term := range searchTerms {
		if alreadyProcessed(term) {
			fmt.Printf("Skipping already processed term: %s\n", term)
			continue
		}

		url := buildWikipediaURL(term)
		fmt.Printf("Scraping: %s\n", url)
		page, err := scrapeWikipedia(url)
		if err != nil {
			log.Printf("Error scraping %s: %v", term, err)
			continue
		}

		err = savePageToDB(page)
		if err != nil {
			log.Printf("Error saving page to DB: %v", err)
			continue
		}
		markAsProcessed(term)

	}

}

func alreadyProcessed(term string) bool {
	var exists bool
	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM processed_searches WHERE search_term = $1)", term).Scan(&exists)
	if err != nil {
		log.Printf("Error checking processed term: %v", err)
		return false // fall back to processing
	}
	return exists
}

func markAsProcessed(term string) {
	_, err := db.Exec("INSERT INTO processed_searches (search_term) VALUES ($1) ON CONFLICT DO NOTHING", term)
	if err != nil {
		log.Printf("Error marking term as processed: %v", err)
	}
}


func extractSearchTerms(logPath string) []string {
	file, err := os.Open(logPath)
	if err != nil {
		log.Printf("Could not open log: %v", err)
		return nil
	}
	defer file.Close()

	re := regexp.MustCompile(`query="([^"]+)"`)
	termsMap := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		match := re.FindStringSubmatch(line)
		if len(match) > 1 {
			term := strings.ToLower(strings.TrimSpace(match[1]))
			termsMap[term] = true

			fmt.Printf("Extracted search term: %s\n", term)
		}
	}

	var terms []string
	for k := range termsMap {
		terms = append(terms, k)
	}
	return terms
}

func buildWikipediaURL(term string) string {
	// Konverter til korrekt URL-encoding (fx "farveblind" -> "Farveblindhed")
	// For demo, antag den bare er korrekt formatteret
	term = strings.ReplaceAll(term, " ", "_")
	return fmt.Sprintf("https://da.wikipedia.org/wiki/%s", strings.Title(term))
}


func scrapeWikipedia(url string) (Page, error) {
	c := colly.NewCollector(
		colly.AllowedDomains("da.wikipedia.org"),
	)

	var page Page
	page.URL = url
	var statusCode int

	// Få HTTP statuskoden
	c.OnResponse(func(r *colly.Response) {
		statusCode = r.StatusCode
	})

	c.OnHTML("#firstHeading", func(e *colly.HTMLElement) {
		page.Title = e.Text
		fmt.Printf("Scraped title: %s\n", page.Title)
	})

	c.OnHTML("div.mw-parser-output", func(e *colly.HTMLElement) {
		text := ""
		e.ForEach("p", func(_ int, el *colly.HTMLElement) {
			text += el.Text + "\n"
		})
		page.Content = text
	})

	// Besøg URL'en
	err := c.Visit(url)
	if err != nil {
		return page, err
	}

	// Tjek for 404 eller anden fejl
	if statusCode == 404 {
		return page, fmt.Errorf("page not found (404)")
	}

	return page, nil
}



func savePageToDB(page Page) error {
	if page.Title == "" || page.URL == "" || page.Content == "" {
		return fmt.Errorf("invalid page data")
	}

	_, err := db.Exec(`
		INSERT INTO pages (url, title, content, language, last_updated)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (url) DO UPDATE
		SET title = EXCLUDED.title,
		    content = EXCLUDED.content,
		    last_updated = NOW()
	`, page.URL, page.Title, page.Content, "da")
	if err != nil {
		return fmt.Errorf("error inserting or updating page: %v", err)
	}

	log.Printf("Saved page to DB (inserted or updated): %s", page.Title)
	return nil
}




