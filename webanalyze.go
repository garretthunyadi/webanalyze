package webanalyze

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// AppDefs provides access to the unmarshalled apps.json file
var AppDefs *AppsDefinition

// Match type encapsulates the App information from a match on a document
type Match struct {
	App     `json:"app"`
	AppName string     `json:"app_name"`
	Matches [][]string `json:"matches"`
	Version string     `json:"version"`
}

func (m *Match) updateVersion(version string) {
	if version != "" {
		m.Version = version
	}
}

// Analyze do http request (if necessary) and analyze response
func Analyze(url string, content *http.Response) ([]Match, error) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(content.Body)
	body := []byte(buf.String())

	cookies := content.Cookies()
	headers := content.Header

	var apps = make([]Match, 0)
	var err error

	var cookiesMap = make(map[string]string)
	for _, c := range cookies {
		cookiesMap[c.Name] = c.Value
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for appname, app := range AppDefs.Apps {
		// TODO: Reduce complexity in this for-loop by functionalising out
		// the sub-loops and checks.

		findings := Match{
			App:     app,
			AppName: appname,
			Matches: make([][]string, 0),
		}

		// check raw html
		if m, v := findMatches(string(body), app.HTMLRegex); len(m) > 0 {
			findings.Matches = append(findings.Matches, m...)
			findings.updateVersion(v)
		}

		// check response header
		headerFindings, version := app.FindInHeaders(headers)
		findings.Matches = append(findings.Matches, headerFindings...)
		findings.updateVersion(version)

		// check url
		if m, v := findMatches(url, app.URLRegex); len(m) > 0 {
			findings.Matches = append(findings.Matches, m...)
			findings.updateVersion(v)
		}

		// check script tags
		doc.Find("script").Each(func(i int, s *goquery.Selection) {
			if script, exists := s.Attr("src"); exists {
				if m, v := findMatches(script, app.ScriptRegex); len(m) > 0 {
					findings.Matches = append(findings.Matches, m...)
					findings.updateVersion(v)
				}
			}
		})

		// check meta tags
		for _, h := range app.MetaRegex {
			selector := fmt.Sprintf("meta[name='%s']", h.Name)
			doc.Find(selector).Each(func(i int, s *goquery.Selection) {
				content, _ := s.Attr("content")
				if m, v := findMatches(content, []AppRegexp{h}); len(m) > 0 {
					findings.Matches = append(findings.Matches, m...)
					findings.updateVersion(v)
				}
			})
		}

		// check cookies
		for _, c := range app.CookieRegex {
			if _, ok := cookiesMap[c.Name]; ok {

				// if there is a regexp set, ensure it matches.
				// otherwise just add this as a match
				if c.Regexp != nil {

					// only match single AppRegexp on this specific cookie
					if m, v := findMatches(cookiesMap[c.Name], []AppRegexp{c}); len(m) > 0 {
						findings.Matches = append(findings.Matches, m...)
						findings.updateVersion(v)
					}

				} else {
					findings.Matches = append(findings.Matches, []string{c.Name})
				}
			}

		}

		if len(findings.Matches) > 0 {
			apps = append(apps, findings)

			// handle implies
			for _, implies := range app.Implies {
				for implyAppname, implyApp := range AppDefs.Apps {
					if implies != implyAppname {
						continue
					}

					f2 := Match{
						App:     implyApp,
						AppName: implyAppname,
						Matches: make([][]string, 0),
					}
					apps = append(apps, f2)
				}

			}
		}
	}

	return apps, nil
}

// runs a list of regexes on content
func findMatches(content string, regexes []AppRegexp) ([][]string, string) {
	var m [][]string
	var version string

	for _, r := range regexes {
		matches := r.Regexp.FindAllStringSubmatch(content, -1)
		if matches == nil {
			continue
		}

		m = append(m, matches...)

		if r.Version != "" {
			version = findVersion(m, r.Version)
		}

	}
	return m, version
}

// parses a version against matches
func findVersion(matches [][]string, version string) string {
	/*
		log.Printf("Matches: %v", matches)
		log.Printf("Version: %v", version)
	*/

	var v string

	for _, matchPair := range matches {
		// replace backtraces (max: 3)
		for i := 1; i <= 3; i++ {
			bt := fmt.Sprintf("\\%v", i)
			if strings.Contains(version, bt) && len(matchPair) >= i {
				v = strings.Replace(version, bt, matchPair[i], 1)
			}
		}

		// return first found version
		if v != "" {
			return v
		}

	}

	return ""
}
