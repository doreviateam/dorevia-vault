package webhooks

import (
	"strings"
)

// ParseWebhookURLs parse les URLs webhooks depuis la configuration
// Format: "event1:url1,url2|event2:url3"
func ParseWebhookURLs(config string) map[string][]string {
	result := make(map[string][]string)
	
	if config == "" {
		return result
	}
	
	// Séparer par |
	events := strings.Split(config, "|")
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		
		// Séparer event_type et URLs
		parts := strings.SplitN(event, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		eventType := strings.TrimSpace(parts[0])
		urlsStr := strings.TrimSpace(parts[1])
		
		if eventType == "" || urlsStr == "" {
			continue
		}
		
		// Séparer les URLs par virgule
		urls := strings.Split(urlsStr, ",")
		urlList := make([]string, 0, len(urls))
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url != "" {
				urlList = append(urlList, url)
			}
		}
		
		if len(urlList) > 0 {
			result[eventType] = urlList
		}
	}
	
	return result
}

