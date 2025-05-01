package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

func searchHandler(w http.ResponseWriter, r *http.Request) {
	//Henter search-query fra URL-parameteren.
	log.Println("Search handler called")

	queryParam := strings.TrimSpace(r.URL.Query().Get("q"))
	if queryParam == "" {
		http.Error(w, "No search query provided", http.StatusBadRequest)
		return
	}
	//TO LOG THE QUERY//
	log.Printf("Search query: %q from %s", queryParam, r.RemoteAddr)
	searchLogger.Printf("query=%q from=%s", queryParam, r.RemoteAddr)

	//Nuild search against Elasticsearch
	pages, err := searchPagesInEs(queryParam)
	if err != nil {
		log.Printf("Error searching Elasticsearch: %v", err)
		http.Error(w, "Error during search", http.StatusInternalServerError)
		return
	}

	// Build search results from Elasticsearch response
	var searchResults []map[string]string
	for _, page := range pages {
		searchResults = append(searchResults, map[string]string{
			"title":       page.Title,
			"url":         page.URL,
			"description": page.Content,
		})
	}

	tmpl, err := template.ParseFiles(templatePath + "search.html")
	if err != nil {
		http.Error(w, "Error loading search template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, map[string]interface{}{
		"Query":   queryParam,
		"Results": searchResults,
	}); err != nil {
		log.Printf("Error executing search template: %v", err)
		http.Error(w, "Error rengering search results", http.StatusInternalServerError)
	}
}

func searchPagesInEs(query string) ([]Page, error) {
	///// TESTS FALLBACK ///////////
	if esClient == nil {
		// Simple DB search for test mode
		var pages []Page
		sqlStmt := "SELECT title, url, content FROM pages WHERE content LIKE ?"
		likeQ := "%" + query + "%"
		rows, err := db.Query(sqlStmt, likeQ)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var p Page
			if err := rows.Scan(&p.Title, &p.URL, &p.Content); err != nil {
				continue
			}
			pages = append(pages, p)
		}
		return pages, nil
	}
	/////// PRODUCTION: real Elasticsearch search ───────────────────────────
	var pages []Page

	searchBody := strings.NewReader(fmt.Sprintf(`{
		"query": {
			"multi_match": {
				"query": "%s",
				"fields": ["title^3", "url^2", "content"]
			}
		}
	}`, query))

	res, err := esClient.Search(
		esClient.Search.WithContext(context.Background()),
		esClient.Search.WithIndex("pages"),
		esClient.Search.WithBody(searchBody),
		esClient.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return pages, err
	}
	defer res.Body.Close()

	var r struct {
		Hits struct {
			Hits []struct {
				Source Page `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return pages, err
	}

	for _, hit := range r.Hits.Hits {
		pages = append(pages, hit.Source)
	}

	return pages, nil
}

func syncPagesToElasticsearch() error {
	rows, err := db.Query("SELECT title, url, content FROM pages")
	if err != nil {
		return fmt.Errorf("error querying pages from DB: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var page Page
		if err := rows.Scan(&page.Title, &page.URL, &page.Content); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		doc, err := json.Marshal(page)
		if err != nil {
			log.Printf("Error marshaling page: %v", err)
			continue
		}

		// Index the document without specifying a document type
		_, err = esClient.Index(
			"pages",                            // Index name
			strings.NewReader(string(doc)),     // JSON document
			esClient.Index.WithRefresh("true"), // Refresh immediately
		)
		if err != nil {
			log.Printf("Error indexing page to ES: %v", err)
			continue
		}

		count++
	}
	log.Printf("Synced %d pages to Elasticsearch", count)
	return nil
}
