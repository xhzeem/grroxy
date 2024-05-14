package proxy

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/glitchedgitz/grroxy-db/base"
	"github.com/glitchedgitz/grroxy-db/templates/actions"
	"github.com/glitchedgitz/grroxy-db/types"
	"github.com/projectdiscovery/dsl"
)

type Store_Req struct {
	ID      string `db:"id" json:"id"`
	Request string `db:"req" json:"req"`
}
type Store_Resp struct {
	Response string `db:"resp" json:"resp"`
}

// MatchReplaceRequest strings or regex
func (p *Proxy) MatchReplaceResponse(resp string) string {
	// resp.ContentLength = 0

	m := make(map[string]interface{})
	m["resp"] = resp
	if v, err := dsl.EvalExpr(p.options.ResponseMatchReplaceDSL, m); err != nil {
		return resp
	} else {
		return fmt.Sprint(v)
	}
}

func (p *Proxy) OnResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {

	if strings.Contains(resp.Request.URL.Host, "grroxy") {
		return resp
	}

	log.Print("[OnResponse] Starting OnResponse")
	userdata := ctx.UserData.(types.UserData)
	if userdata.Action == "drop" {
		log.Printf("[Response][Intercept][%s]: Dropping Response because request dropped \n", userdata.ID)
		return DropReqResp(ctx.Req)
	}

	userdata.Action = ""

	id := userdata.ID
	userdata.HasResp = true

	if resp == nil {
		log.Print("[OnResponse]Returning nil response")
		return nil
	}

	responseInBytes, err := base.ResponseToByte(resp)
	base.CheckErr("[OnResponse]", err)
	responseInString := string(responseInBytes)

	var title string
	if responseInBytes != nil {
		title, _ = base.ExtractTitle(responseInBytes)
	}

	userdata.Resp = types.ResponseData{
		HasCookies: len(resp.Cookies()) > 0,
		Title:      title,
		Mime:       resp.Header.Get("content-type"),
		Status:     resp.StatusCode,
		Length:     len(responseInString),
		Date:       resp.Header.Get("Date"),
		Time:       time.Now().Format(time.RFC3339),
	}

	r_data := Store_Resp{
		Response: responseInString,
	}

	p.grroxydb.Update("_raw", id, r_data)

	// var updatedString string
	var edited bool
	// Intercept
	if p.options.Intercept && p.checkFilters(userdata) {

		responseInString, edited = p.interceptWait(&userdata, "resp", resp.ContentLength)

		if userdata.Action == "drop" {
			log.Println("[Response][Intercept][%s]: Dropping Response \n", userdata.Host+"/"+userdata.Req.Path)
			ctx.UserData = userdata
			return DropReqResp(ctx.Req)
		}

		if edited {
			userdata.IsRespEdited = true
		}

		p.grroxydb.Update("_data", userdata.ID, userdata)

		base.CheckErr("Error in reading updated request", err)

	}

	p._responseAddToDB(userdata)
	resp, err = http.ReadResponse(bufio.NewReader(strings.NewReader(fmt.Sprint(responseInString))), ctx.Req)
	base.CheckErr("[onResponse]: ", err)
	ctx.UserData = userdata
	// defer resp.Body.Close()
	return resp
}

func (p *Proxy) _responseAddToDB(userdata types.UserData) {
	userdata.Resp.Mime = strings.ToLower(userdata.Resp.Mime)
	userdata.Resp.Mime = strings.ReplaceAll(userdata.Resp.Mime, "\"", "")
	userdata.Resp.Mime = strings.ReplaceAll(userdata.Resp.Mime, "'", "")
	userdata.Resp.Mime = strings.ReplaceAll(userdata.Resp.Mime, " ", "")

	p.DBUpdate("_data", userdata.ID, userdata)

	tmpdata := types.UserData{
		Resp: userdata.Resp,
	}

	d := base.StructToMap(&tmpdata, "json")
	results, _ := p.templates.Run(d, "proxy:response")

	log.Println("[_requestAddToDB] Checking template results: ", results)

	for _, y := range results {

		name := y.Data["name"].(string)

		if len(name) == 0 {
			continue
		}

		var l_data = types.Label{
			Name:  name,
			Color: y.Data["color"].(string),
			Type:  y.Data["type"].(string),
			ID:    userdata.ID,
		}

		switch y.ActionName {
		case actions.CreateLabel:
			p.DBAttachLabel(l_data)
		default:
			log.Println("[_requestAddToDB] Unknown Action")
		}
	}
}
