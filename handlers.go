package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nleeper/goment"
)

func indexHandler(store *proxyStore) func(w http.ResponseWriter, r *http.Request) {
	const indexTmpl = `<!doctype="html">
<meta charset="UTF-8">
<head>
	<style>
		body {
			background-color: #f9f9f9;
			color: #404040;
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol";
			font-weight: 300;
			line-height: 2;
			margin-left: auto;
			margin-right: auto;
			margin-top: 4rem;
			max-width: 50rem;
		}
		h1 {
			font-size: 1.4rem;
			font-weight: 300;
			margin: 0;
		}
		span {
			background: #efefef;
			color: #606060;
			font-family: monospace, monospace;
			font-size: 1em;
			padding: 0.3em;
		}
		a {
			color: #222;
			border-bottom: 1px solid #c2c2c2;
			padding-bottom: 0.2rem;
			text-decoration: none;
		}
		a:visited {
			color: #222;
		}
		a:hover {
			color: #000;
		}
		
		.event {
			border: solid 1px #ddd;
			padding: 1rem;
			margin-bottom: 2rem;
			overflow-wrap: break-word;
		}
		
		.event-top {
			padding-top: 5px;
			padding-bottom: 5px;
			border-bottom: solid 1px #ddd;
		}
		.event-content {
			padding-top: 5px;
			padding-bottom: 5px;
			color: #202020;
		}
		
		.paging {
			display: flex;
			flex-direction: row-reverse;
		}

		.paging > div {
			padding: 3px;
		}

		hr {
			border: solid 1px #ddd;
			margin-top: 4rem;
		}

	</style>
</head>
<body>
	<h2>nostr relay proxy</h2>
	<div>
		{{$_ := ""}}
		{{$e := ""}}
		{{ range $_, $e = .Events }}
			<a href="{{ .Permalink }}">
				<div class="event">
					<div class="event-top">
						{{ $e.AuthorName }} {{ $e.KindName }} {{ $e.At }}
					</div>
					<div class="event-content">
						{{ $e.Event.Content }}
					</div>
				</div>
			</a>
		{{ end }}
	</div>

	<div class="paging">
		<div><a href="?until={{ $e.Event.CreatedAt}}">next</a></div>
		<div><a href="?since={{ $e.Event.CreatedAt}}">prev</a></div>
	</div>

	<hr>

	<div>
		<span>rendered in {{ .Elapsed }}</span>
	</div>
</body>
`
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var (
			since = r.URL.Query().Get("since")
			until = r.URL.Query().Get("until")
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		events, err := store.db.QueryEvents(ctx, nostr.Filter{
			Kinds: []int{1, 6, 7},
			Limit: 25,
			Since: parseNostrTimestamp(since),
			Until: parseNostrTimestamp(until),
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		data := struct {
			Elapsed string
			Events  []uiEvent
		}{}
		for event := range events {
			kindName := ""
			switch event.Kind {
			case 1:
				kindName = "posted"
			case 6:
				kindName = "reposted"
			case 7:
				kindName = "reacted"
			case 9734:
				kindName = "zapped"
			default:
				kindName = fmt.Sprintf("kind %d", event.Kind)
			}

			authorName := event.PubKey[:8] + ":" + event.PubKey[len(event.PubKey)-8:]
			{
				ctx, cancel := context.WithTimeout(ctx, time.Millisecond*200)
				defer cancel()
				events, err := store.db.QueryEvents(ctx, nostr.Filter{
					Kinds:   []int{0},
					Authors: []string{event.PubKey},
					Limit:   1,
				})
				if err == nil {
					for e := range events {
						// {name: <username>, about: <string>, picture: <url, string>}
						var m map[string]string
						if err := json.Unmarshal([]byte(e.Content), &m); err == nil {
							authorName = m["name"]
						}
					}
				}
			}

			at, _ := goment.New(event.CreatedAt.Time())
			fromNow := at.FromNow()

			permalink := ""
			nevent, err := nip19.EncodeEvent(event.ID, []string{}, event.PubKey)
			if err == nil {
				permalink = "https://njump.me/" + nevent
			}

			data.Events = append(data.Events, uiEvent{
				Event:      event,
				KindName:   kindName,
				AuthorName: authorName,
				At:         fromNow,
				Permalink:  permalink,
			})
		}

		t, err := template.New("index").Parse(indexTmpl)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		data.Elapsed = time.Since(start).String()

		w.Header().Add("content-type", "text/html")
		t.Execute(w, data)
	}
}

type uiEvent struct {
	Event      *nostr.Event
	KindName   string
	AuthorName string
	At         string
	Permalink  string
}

func parseNostrTimestamp(s string) *nostr.Timestamp {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}

	ts := nostr.Timestamp(i)
	return &ts
}
