package sites

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"app/main.go/internal/models/domain"

	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
)

// ScrapeLococlub — скрапер для сайта lococlub.es
func ScrapeLococlub(ctx context.Context, baseURL string, shutdownChan <-chan struct{}) ([]domain.Event, error) {
	var events []domain.Event
	var eventLinks []string
	var mu sync.Mutex

	// 1. Собираем ссылки на все события
	collectLinksGez := geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{baseURL},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			r.HTMLDoc.Find("article.mec-event-article a.mec-color-hover").Each(func(i int, sel *goquery.Selection) {
				if href, ok := sel.Attr("href"); ok {
					absoluteURL, err := r.Request.URL.Parse(href)
					if err == nil {
						mu.Lock()
						eventLinks = append(eventLinks, absoluteURL.String())
						mu.Unlock()
					}
				}
			})
		},
	})
	collectLinksGez.Start()
	eventLinks = uniqueStrings(eventLinks)

	// 2. Для каждой ссылки собираем детали
	for _, link := range eventLinks {
		select {
		case <-ctx.Done():
			return events, ctx.Err()
		case <-shutdownChan:
			return events, fmt.Errorf("shutdown")
		default:
			event, err := scrapeLococlubEventDetails(link)
			if err != nil {
				continue
			}
			event.Status = domain.EventStatusNew
			events = append(events, event)
		}
	}

	return events, nil
}

func scrapeLococlubEventDetails(url string) (domain.Event, error) {
	var event domain.Event
	event.EventLink = url

	gez := geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{url},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			// Название
			name := r.HTMLDoc.Find("h1.mec-single-title").Text()
			if name == "" {
				name = r.HTMLDoc.Find(".mec-single-event-title").Text()
			}
			event.Name = strings.TrimSpace(name)
			event.Name = strings.ReplaceAll(event.Name, "\n", "")
			event.Name = strings.ReplaceAll(event.Name, "\t", "")

			// Описание
			descSelection := r.HTMLDoc.Find(".mec-single-event-description").Clone()
			descSelection.Find("script").Remove()
			event.Description = strings.TrimSpace(descSelection.Text())
			event.Description = strings.ReplaceAll(event.Description, "\t", "")
			event.Description = strings.ReplaceAll(event.Description, "\n", "")

			// Фото
			if src, ok := r.HTMLDoc.Find(".mec-events-event-image img").Attr("src"); ok {
				event.Photo = src
			}

			// Дата и время
			dateStr := strings.TrimSpace(r.HTMLDoc.Find(".mec-single-event-date .mec-start-date-label").Text())
			timeStr := strings.TrimSpace(r.HTMLDoc.Find(".mec-single-event-time .mec-events-abbr").First().Text())

			if dateStr != "" {
				t, err := time.Parse("02 Jan 2006", dateStr)
				if err == nil {
					event.Date = t
					if timeStr != "" {
						parts := strings.Split(timeStr, ":")
						if len(parts) == 2 {
							hour, _ := strconv.Atoi(parts[0])
							min, _ := strconv.Atoi(parts[1])
							event.Date = time.Date(t.Year(), t.Month(), t.Day(), hour, min, 0, 0, t.Location())
						}
					}
				}
			}

			// Цена
			priceText := r.HTMLDoc.Find(".mec-event-cost").Text()
			if priceText == "" {
				priceText = r.HTMLDoc.Find("dd.mec-events-event-cost").Text()
			}

			if priceText != "" {
				re := regexp.MustCompile(`\d+(?:[.,]\d+)?`)
				matches := re.FindAllString(priceText, -1)

				var minPrice float64
				first := true

				for _, m := range matches {
					m = strings.ReplaceAll(m, ",", ".")
					p, err := strconv.ParseFloat(m, 64)
					if err == nil {
						if first || p < minPrice {
							minPrice = p
							first = false
						}
					}
				}

				if !first {
					event.Price = minPrice
					event.Currency = "EUR"
				}
			}

			// Ссылка на покупку
			if buyLink, ok := r.HTMLDoc.Find(".mec-booking-button").Attr("href"); ok {
				event.Description += "\n\nКупить билет: " + buyLink
			}

			// Видео (YouTube, Vimeo iframes или video tags)
			if src, ok := r.HTMLDoc.Find(".mec-single-event-description iframe[src*=\"youtube\"]").Attr("src"); ok {
				event.VideoURL = src
			} else if src, ok := r.HTMLDoc.Find(".mec-single-event-description iframe[src*=\"vimeo\"]").Attr("src"); ok {
				event.VideoURL = src
			} else if src, ok := r.HTMLDoc.Find(".mec-single-event-description video source").Attr("src"); ok {
				event.VideoURL = src
			} else if src, ok := r.HTMLDoc.Find(".mec-single-event-description video").Attr("src"); ok {
				event.VideoURL = src
			}

			event.Tag = ""
		},
	})
	gez.Start()

	return event, nil
}

// uniqueStrings удаляет дубликаты из slice строк
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, v := range input {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
