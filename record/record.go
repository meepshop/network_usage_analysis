package record

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

/* Record example
{
	"httpRequest": {
		"cacheHit": true,
		"cacheLookup": true,
		"referer": "https://www.pimgo.com.tw/pages/products?offset=60",
		"remoteIp": "223.136.88.46",
		"requestMethod": "GET",
		"requestSize": "82",
		"requestUrl": "https://gc.meepcloud.com/meepshop/f00e5170-6609-4e9c-8f7b-a16a47379a39/files/920edce1-49f3-4dee-8476-2de6575b3637.jpeg?w=640",
		"responseSize": "100262",
		"status": 200,
		"userAgent": "Mozilla/5.0 (iPhone; CPU iPhone OS 11_2_5 like Mac OS X) AppleWebKit/604.5.6 (KHTML, like Gecko) Mobile/15D60 [FBAN/FBIOS;FBAV/169.0.0.50.95;FBBV/104829965;FBDV/iPhone10,5;FBMD/iPhone;FBSN/iOS;FBSV/11.2.5;FBSS/3;FBCR/&#20013-&#33775-&#38651-&#20449-;FBID/phone;FBLC/zh_TW;FBOP/5;FBRV/0]"
	},
	"insertId": "dlqob9g13q7drd",
	"jsonPayload": {
		"@type": "type.googleapis.com/google.cloud.loadbalancing.type.LoadBalancerLogEntry",
		"statusDetails": "response_from_cache"
	},
	"logName": "projects/instant-matter-785/logs/requests",
	"receiveTimestamp": "2018-05-01T01:00:01.310484094Z",
	"resource": {
		"labels": {
			"backend_service_name": "",
			"forwarding_rule_name": "lb-resizer-forwarding-rule-2",
			"project_id": "instant-matter-785",
			"target_proxy_name": "lb-resizer-target-proxy-gc-july",
			"url_map_name": "lb-resizer",
			"zone": "global"
		},
		"type": "http_load_balancer"
	},
	"severity": "INFO",
	"spanId": "8d2bfeee77f9442a",
	"timestamp": "2018-05-01T00:59:59.965472977Z",
	"trace": "projects/instant-matter-785/traces/1291b46b2a3b8e05c6d06e846a3433d5"
}
*/
type Record struct {
	HTTPRequest HTTPRequest `json:"httpRequest"`
	InsertID    string      `json:"insertId"`
	JSONPayload struct {
		Type          string `json:"@type"`
		StatusDetails string `json:"statusDetails"`
	} `json:"jsonPayload"`
	LogName          string   `json:"logName"`
	ReceiveTimestamp string   `json:"receiveTimestamp"`
	Resource         Resource `json:"resource"`
	Severity         string   `json:"severity"`
	SpanID           string   `json:"spanId"`
	Timestamp        string   `json:"timestamp"`
	Trace            string   `json:"trace"`
}

type HTTPRequest struct {
	CacheHit      bool   `json:"cacheHit"`
	CacheLookup   bool   `json:"cacheLookup"`
	Referer       string `json:"referer"`
	RemoteIP      string `json:"remoteIp"`
	RequestMethod string `json:"requestMethod"`
	RequestSize   string `json:"requestSize"`
	RequestURL    string `json:"requestUrl"`
	ResponseSize  string `json:"responseSize"`
	Status        int    `json:"status"`
	UserAgent     string `json:"userAgent"`
}

type Resource struct {
	Labels struct {
		BackendServiceName string `json:"backend_service_name"`
		ForwardingRuleName string `json:"forwarding_rule_name"`
		ProjectID          string `json:"project_id"`
		TargetProxyName    string `json:"target_proxy_name"`
		URLMapName         string `json:"url_map_name"`
		Zone               string `json:"zone"`
	} `json:"labels"`
	Type string `json:"type"`
}

var regexps = map[string][]string{
	"old": []string{
		// product
		"^(https?:\\/\\/gc.meepcloud.com\\/meepshop\\/shop\\/)([A-Za-z0-9-_]{1,})\\/products\\/([A-Za-z0-9]{1,})\\/(original|gallery)",
		// layout、background、favicon、logo、activity
		"^(https?:\\/\\/gc.meepcloud.com\\/meepshop\\/shop\\/)([A-Za-z0-9-_]{1,})\\/(layout|background|favicon|logo|activity)\\/",
	},
	"new": []string{
		// product
		"^(https?:\\/\\/gc.meepcloud.com\\/meepshop\\/)([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})\\/files\\/([a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}.)",
	},
}

func (r *Record) ParseToThroughputItem() (ThroughputItem, error) {

	ti := ThroughputItem{}

	system := ""
	url := strings.Replace(r.HTTPRequest.RequestURL, "%2F", "/", -1)
	urlByte := []byte(url)

systemCheckLoop:
	for v, rs := range regexps {
		for _, r := range rs {
			if match, _ := regexp.Match(r, urlByte); match {
				system = v
				break systemCheckLoop
			}
		}
	}

	if system == "old" {
		s := strings.Split(url, "/")
		ti.cname = s[5]
	} else if system == "new" {
		s := strings.Split(url, "/")
		ti.storeid = s[4]
	} else {
		// log.Println(url)
		return ti, errors.New("ParseToThroughputItem RequestURL not match. URL:" + r.HTTPRequest.RequestURL)
	}

	ti.system = system
	ti.requestsize, _ = strconv.ParseInt(r.HTTPRequest.RequestSize, 10, 64)
	ti.responsesize, _ = strconv.ParseInt(r.HTTPRequest.ResponseSize, 10, 64)

	return ti, nil
}
