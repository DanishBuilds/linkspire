// services/workers/internal/scraper/scraper.go
package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	"github.com/linkspire/linkspire/pkg/ssrfguard"
	"github.com/linkspire/linkspire/services/workers/internal/store"
)

type Worker struct {
	Store  *store.Store
	Client *http.Client
	Log    *slog.Logger
}

func New(st *store.Store, log *slog.Logger) *Worker {
	return &Worker{Store: st, Client: ssrfguard.NewClient(ssrfguard.DefaultConfig()), Log: log}
}

func (w *Worker) Tick(ctx context.Context) {
	links, err := w.Store.LinksNeedingMetadata(ctx, 20)
	if err != nil {
		w.Log.Error("scrape list", "err", err)
		return
	}
	for _, l := range links {
		if err := w.scrapeOne(ctx, l); err != nil {
			w.Log.Warn("scrape", "code", l.Code, "err", err)
		}
	}
}

func (w *Worker) scrapeOne(ctx context.Context, l store.LinkRow) error {
	u, err := url.Parse(l.Destination)
	if err != nil {
		return err
	}
	if err := ssrfguard.ValidateURLScheme(u.Scheme); err != nil {
		return err
	}
	robotsOK := w.robotsAllowed(ctx, u)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.Destination, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "LinkspireBot/1.0 (+https://linkspire.io/bot)")
	resp, err := w.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	title, desc, image, favicon, canonical, og, tw, ld := extract(body, l.Destination)
	return w.Store.UpsertMetadata(ctx, l, title, desc, image, favicon, canonical, resp.Request.URL.String(), resp.StatusCode, robotsOK, og, tw, ld)
}

func (w *Worker) robotsAllowed(ctx context.Context, u *url.URL) bool {
	robotsURL := u.Scheme + "://" + u.Host + "/robots.txt"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return true
	}
	resp, err := w.Client.Do(req)
	if err != nil {
		return true // fail-open if robots.txt unreachable
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	return parseRobots(string(b), u.Path)
}

func parseRobots(body, path string) bool {
	uaStar := false
	disallow := []string{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		low := strings.ToLower(line)
		switch {
		case strings.HasPrefix(low, "user-agent:"):
			v := strings.TrimSpace(line[len("User-agent:"):])
			uaStar = v == "*" || strings.EqualFold(v, "LinkspireBot")
		case uaStar && strings.HasPrefix(low, "disallow:"):
			v := strings.TrimSpace(line[len("Disallow:"):])
			if v != "" {
				disallow = append(disallow, v)
			}
		}
	}
	for _, d := range disallow {
		if d == "/" || strings.HasPrefix(path, d) {
			return false
		}
	}
	return true
}

func extract(body []byte, base string) (title, desc, image, favicon, canonical string, og, tw, ld []byte) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return
	}
	ogm := map[string]string{}
	twm := map[string]string{}
	var jsonld []json.RawMessage
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if title == "" && n.FirstChild != nil {
					title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				k, v := attr(n, "property"), attr(n, "content")
				if k == "" {
					k = attr(n, "name")
				}
				switch strings.ToLower(k) {
				case "og:title":
					ogm["title"] = v
					if title == "" {
						title = v
					}
				case "og:description", "description":
					ogm["description"] = v
					if desc == "" {
						desc = v
					}
				case "og:image":
					ogm["image"] = v
					if image == "" {
						image = v
					}
				case "twitter:card", "twitter:title", "twitter:description", "twitter:image":
					twm[k] = v
				}
			case "link":
				rel := strings.ToLower(attr(n, "rel"))
				href := attr(n, "href")
				if strings.Contains(rel, "icon") && favicon == "" {
					favicon = href
				}
				if rel == "canonical" {
					canonical = href
				}
			case "script":
				if strings.EqualFold(attr(n, "type"), "application/ld+json") && n.FirstChild != nil {
					raw := strings.TrimSpace(n.FirstChild.Data)
					if json.Valid([]byte(raw)) {
						jsonld = append(jsonld, json.RawMessage(raw))
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	og, _ = json.Marshal(ogm)
	tw, _ = json.Marshal(twm)
	if len(jsonld) > 0 {
		ld, _ = json.Marshal(jsonld)
	}
	_ = base
	return
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}
