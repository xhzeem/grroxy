package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/elazarl/goproxy"
	"github.com/glitchedgitz/grroxy-db/base"
	"github.com/glitchedgitz/grroxy-db/sdk"
	"github.com/glitchedgitz/grroxy-db/types"
	"github.com/projectdiscovery/dsl"
)

// MatchReplaceRequest strings or regex
func (p *Proxy) MatchReplaceRequest(req string) string {
	// lazy mode - ninja level - elaborate
	m := make(map[string]interface{})
	m["request"] = req
	if v, err := dsl.EvalExpr(p.options.RequestMatchReplaceDSL, m); err != nil {
		return req
	} else {
		return fmt.Sprint(v)
	}
}

// const CONCURRENCY = 20

// var rateLimit = make(chan string, CONCURRENCY)

func (p *Proxy) OnRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	// rateLimit <- ""
	// defer func() { <-rateLimit }()
	var userdata types.UserData

	if ctx.UserData != nil {
		userdata = ctx.UserData.(types.UserData)
	}

	// Convert request to string
	requestInBytes, err := httputil.DumpRequestOut(req, true)
	base.CheckErr("Req:Dumping Bytes Error", err)
	requestInString := string(requestInBytes)

	// Initiate variables
	var (
		id      = base.RandomString(15)
		method  = "GET"
		host    = req.URL.Host
		port    = ""
		isHttps = req.TLS != nil
	)

	// Set method
	func() {
		if req.Method != "" {
			method = req.Method
		}
	}()

	// Set host and port
	func() {
		if strings.Contains(host, ":") {
			t := strings.Split(host, ":")
			host = t[0]
			port = t[1]
		}

		if isHttps {
			host = "https://" + host
		} else {
			host = "http://" + host
		}
	}()

	userdata = types.UserData{
		ID:          id,
		StoreID:     id,
		Host:        host,
		Port:        port,
		IP:          req.RemoteAddr,
		HasResponse: false,
		OriginalRequest: types.RequestData{
			Method:        method,
			HasCookies:    len(req.Cookies()) > 0,
			HasParams:     len(req.URL.Query()) > 0,
			ContentLength: len(requestInString),
			IsHTTPS:       isHttps,
			Url:           req.URL.RequestURI(),
			Path:          req.URL.Path,
			Query:         req.URL.RawQuery,
			Fragment:      req.URL.RawFragment,
		},
		OriginalResponse: types.ResponseData{
			Title:         "",
			Mimetype:      "",
			StatusCode:    0,
			ContentLength: 0,
			HasCookies:    false,
			Date:          "",
			Time:          "",
		},
		IsRequestEdited:  false,
		IsResponseEdited: false,
		// Labels:           []string{"test"},
	}

	// Add to database
	func() {
		p.DBCreate("store", map[string]string{
			"id":      userdata.ID,
			"request": requestInString,
		})
		// p.grroxydb.Create("data", userdata)
	}()

	// Intercept
	if p.options.Intercept {

		var wg sync.WaitGroup

		wg.Add(1)

		// Add to intercept database
		p.DBCreate("intercept", userdata)

		// Realtime Subscription
		stream, err := sdk.CollectionSet[types.RealtimeRecord](p.grroxydb, "intercept").Subscribe("intercept/" + id)

		base.CheckErr(fmt.Sprintf("[Request][Intercept][%s] Error while creating stream \n", id), err)
		log.Printf("[Request][Intercept][%s]: Subcrbied to the record \n", id)

		<-stream.Ready()
		log.Printf("[Request][Intercept][%s]: Subcrbie is ready\n", id)

		updatedRow := types.RealtimeRecord{}
		action := ""

		// Listening to changes
		for ev := range stream.Events() {
			log.Printf("[Request][Intercept][%s]: %s %v\n", id, ev.Action, ev.Record)

			if ev.Record.Action == "forward" {
				log.Printf("[Request][Intercept][%s]: Forwarding Request\n", id)
				updatedRow = ev.Record
				action = "forward"

				break
			}
			if ev.Record.Action == "drop" { // GPT4's Idea
				log.Printf("[Request][Intercept][%s]: Drop Request\n", id)
				action = "drop"
				break
				// return req, goproxy.NewResponse(req, goproxy.ContentTypeText, 444, "")
			}
		}

		log.Printf("[Request][Intercept][%s]: About to Unsubscribe Request\n", id)
		stream.Unsubscribe()
		log.Printf("[Request][Intercept][%s]: Unsubscribe Request\n", id)

		// Move row from intercept to data
		p.grroxydb.Delete("intercept", userdata.ID)
		if action == "drop" {
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, 444, "")
		}
		collection := sdk.CollectionSet[any](p.grroxydb, "store")
		updatedData, err := collection.One(updatedRow.ID)

		base.CheckErr("Error in getting updated data", err)

		var updatedString string

		// log.Println("Edited Request is not empty -----------------------")
		// log.Println(updatedData)

		upData := updatedData.(map[string]interface{})
		log.Println("Updated Data --------------  ", upData)

		if updatedRow.IsRequestEdited {
			updatedString = upData["request_edited"].(string)
			userdata.IsRequestEdited = true
			// p.DBUpdate("store", userdata.ID, map[string]string{
			// 	"request_edited": updatedString,
			// })
		} else {
			updatedString = upData["request"].(string)
		}

		// Convert string to request
		requestNew, err := http.ReadRequest(bufio.NewReader(strings.NewReader(fmt.Sprint(updatedString))))
		base.CheckErr("Error in reading updated request", err)

		// Todo: Set Host, Port and Scheme
		// req.URL.Host // this can include the port also
		// req.URL.Scheme

		p._requestAddToDB(userdata)
		ctx.UserData = userdata

		req.Body.Close()
		return requestNew, nil
	}

	p._requestAddToDB(userdata)

	ctx.UserData = userdata
	return req, nil
}

func (p *Proxy) _requestAddToDB(userdata types.UserData) {

	p.grroxydb.Create("data", userdata)

	p.DBCreate("sites", map[string]string{
		"site": userdata.Host,
	})

	s_data := types.SitemapGet{
		Host:     userdata.Host,
		Path:     userdata.OriginalRequest.Path,
		Query:    userdata.OriginalRequest.Query,
		Fragment: userdata.OriginalRequest.Fragment,
		Type:     "folder",
		MainID:   userdata.ID,
	}

	// check path and detect file or folder
	// d := strings.Split(s_data.Path, "/")
	// folder := d[len(d)-1]

	// // check if this is file
	// if strings.Contains(folder, ".") {
	// 	s_data.Type = "file"

	// 	// check extension
	// 	e := strings.Split(folder, ".")
	// 	ext := e[len(e)-1]
	// 	switch ext {
	// 	case "js":
	// }

	p.DBNewSitemap(s_data)
}
