package main

import (
	"path"

	save "github.com/glitchedgitz/grroxy-db/save"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type FetchedData struct {
	Request  string
	Response string
}

func (p *Proxy) FetchRequest(uniqID string, host string, port string) FetchedData {
	runtime.LogDebug(p.options.Ctx, uniqID)
	return FetchedData{
		Request:  string(save.ReadFile(path.Join(p.config.ProjectDirectory, "targets", host, port, "req", uniqID))),
		Response: string(save.ReadFile(path.Join(p.config.ProjectDirectory, "targets", host, port, "resp", uniqID))),
	}
}

// type Search struct {
// 	HudID         string   `json:""`
// 	SearchID      string   `json:""`
// 	Keywords      []string `json:""`
// 	IDs           []string `json:""`
// 	MatchAll      bool     `json:""`
// 	CaseSensitive bool     `json:""`
// 	stop          chan bool
// }

// var allSearches map[string]Search

// func (S *Search) matchAll(uniqID string, f func(string) string) bool {
// 	data := f(string(save.ReadFile(uniqID)))
// 	for _, keyword := range S.Keywords {
// 		if !strings.Contains(data, f(keyword)) {
// 			return false
// 		}
// 	}
// 	return true
// }

// func (S *Search) matchAny(uniqID string, f func(string) string) bool {
// 	data := f(string(save.ReadFile(uniqID)))
// 	for _, keyword := range S.Keywords {
// 		if strings.Contains(data, f(keyword)) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (p *Proxy) NewSearch(results []interface{}) {

// 	S := ParseDataFromFrontend[Search](results)

// 	if _, exists := allSearches[S.HudID]; exists {
// 		allSearches[S.HudID].stop <- true
// 	}

// 	allSearches[S.HudID] = S

// 	go func() {
// 		var f1 func(string, func(string) string) bool
// 		var f2 func(string) string

// 		if S.MatchAll {
// 			f1 = S.matchAll
// 		} else {
// 			f1 = S.matchAny
// 		}

// 		if S.CaseSensitive {
// 			f2 = func(s string) string {
// 				return s
// 			}
// 		} else {
// 			f2 = strings.ToLower
// 		}

// 		for _, uniqID := range S.IDs {
// 			if f1(uniqID, f2) {
// 				runtime.EventsEmit(p.options.Ctx, S.SearchID, uniqID)
// 			}
// 		}
// 	}()

// 	<-S.stop
// }
