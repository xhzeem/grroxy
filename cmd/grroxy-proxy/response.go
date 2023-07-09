package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/glitchedgitz/grroxy-db/base"
	"github.com/glitchedgitz/grroxy-db/sdk"
	"github.com/glitchedgitz/grroxy-db/types"
	"github.com/projectdiscovery/dsl"
	"golang.org/x/net/html"
)

type Store_Req struct {
	ID      string `db:"id" json:"id"`
	Request string `db:"request" json:"request"`
}
type Store_Resp struct {
	Response string `db:"response" json:"response"`
}

// MatchReplaceRequest strings or regex
func (p *Proxy) MatchReplaceResponse(resp string) string {
	// resp.ContentLength = 0

	m := make(map[string]interface{})
	m["response"] = resp
	if v, err := dsl.EvalExpr(p.options.ResponseMatchReplaceDSL, m); err != nil {
		return resp
	} else {
		return fmt.Sprint(v)
	}
}

func (p *Proxy) OnResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {

	userdata := ctx.UserData.(types.UserData)
	log.Printf("[Response][Intercept][%s]: ResponseUserdata \n", userdata)

	id := userdata.ID
	userdata.HasResponse = true

	if resp == nil {
		return nil
	}

	responseInBytes, err := base.ResponseToByte(resp)
	base.CheckErr("[OnResponse]", err)
	responseInString := string(responseInBytes)

	var title string
	if responseInBytes != nil {
		title, _ = extractTitle(responseInBytes)
	}

	userdata.OriginalResponse = types.ResponseData{
		HasCookies:    len(resp.Cookies()) > 0,
		Title:         title,
		Mimetype:      resp.Header.Get("content-type"),
		StatusCode:    resp.StatusCode,
		ContentLength: len(responseInString),
		Date:          resp.Header.Get("Date"),
		Time:          time.Now().Format(time.RFC3339),
	}

	r_data := Store_Resp{
		Response: responseInString,
	}

	p.grroxydb.Update("store", id, r_data)

	// Intercept
	if p.options.Intercept {

		var wg sync.WaitGroup

		wg.Add(1)

		//Add to database
		p.DBCreate("intercept", userdata)

		// Realtime Subscription
		stream, err := sdk.CollectionSet[types.RealtimeRecord](p.grroxydb, "intercept").Subscribe("intercept/" + id)

		base.CheckErr(fmt.Sprintf("[Response][Intercept][%s] Error while creating stream \n", id), err)
		log.Printf("[Response][Intercept][%s]: Subcrbied to the record \n", id)

		<-stream.Ready()
		log.Printf("[Response][Intercept][%s]: Subcrbie is ready\n", id)

		updatedRow := types.RealtimeRecord{}
		action := ""
		for ev := range stream.Events() {
			log.Printf("[Response][Intercept][%s]: %s %v\n", id, ev.Action, ev.Record)

			if ev.Record.Action == "forward" {
				log.Printf("[Response][Intercept][%s]: Forwarding Response\n", id)
				updatedRow = ev.Record
				action = "forward"
				break
			}
			if ev.Record.Action == "drop" { // GPT4's Idea
				action = "drop"
				log.Printf("[Response][Intercept][%s]: Drop Response\n", id)
				break
			}
		}

		stream.Unsubscribe()
		log.Printf("[Response][Intercept][%s]: About to Unsubscribe Response\n", id)

		p.grroxydb.Delete("intercept", id)
		p.grroxydb.Create("data", userdata)

		if action == "drop" {
			return goproxy.NewResponse(ctx.Req, goproxy.ContentTypeText, 444, "")
		}

		collection := sdk.CollectionSet[any](p.grroxydb, "store")
		updatedData, err := collection.One(updatedRow.ID)
		if err != nil {
			log.Println(err)
		}

		var updatedString string

		log.Println("[onResponse] Edited Response is not empty -----------------------")
		log.Println(updatedData)

		upData := updatedData.(map[string]interface{})
		log.Println("[onResponse] Updated Data --------------  ", upData)

		if updatedRow.IsResponseEdited {
			updatedString = upData["response_edited"].(string)
			userdata.IsResponseEdited = true
			// p.DBUpdate("store", userdata.ID, map[string]string{
			// 	"response_edited": updatedString,
			// })
		} else {
			updatedString = upData["response"].(string)
		}

		log.Println("[onResponse][ReadResponse]")
		respNew, err := http.ReadResponse(bufio.NewReader(strings.NewReader(fmt.Sprint(updatedString))), ctx.Req)
		// reader := bufio.NewReader(bytes.NewReader([]byte(updatedString)))
		// respNew, err := http.ReadResponse(reader, nil)

		if err != nil {
			log.Println("Error in reading response", err)
		}

		if respNew == nil {
			log.Println("[onResponse] Nil Response", err)
		}

		defer resp.Body.Close()
		ctx.UserData = userdata
		return respNew
	}

	p._responseAddToDB(userdata)
	resp, err = http.ReadResponse(bufio.NewReader(strings.NewReader(fmt.Sprint(responseInString))), ctx.Req)
	base.CheckErr("[onResponse]: ", err)
	ctx.UserData = userdata
	return resp
}

func (p *Proxy) _responseAddToDB(userdata types.UserData) {
	p.DBUpdate("data", userdata.ID, userdata)
}

func extractTitle(respByte []byte) (string, string) {

	title := ""
	favicon := ""

	z := html.NewTokenizer(bytes.NewReader(respByte))

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		t := z.Token()

		if t.Type == html.StartTagToken {
			if t.Data == "title" {
				if z.Next() == html.TextToken {
					title = strings.TrimSpace(z.Token().Data)
					break
				}
			}
			// else if t.Data == "link" {
			// 	if z.Next() == html.TextToken {
			// 		favicon = strings.TrimSpace(z.Token().Data)
			// 		break
			// 	}
			// }
		}
	}
	return title, favicon
}
