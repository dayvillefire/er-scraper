package agent

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/jbuchbinder/shims"
)

var (
	ErrNotAuthorized = errors.New("not authorized")
)

const (
	ContextLogin    = 0
	ContextDownload = 1
)

type Agent struct {
	Debug    bool
	LoginUrl string
	Username string
	Password string

	ContextSwitch int

	reqMap  map[string]network.RequestID
	urlMap  map[string]string
	bodyMap map[string][]byte
	attr    map[string]string
	cookies []*network.Cookie
	ctx     context.Context
	cancel  context.CancelFunc
	cfunc   []context.CancelFunc
	done    chan string

	initialized bool
	cancelled   bool
	wg          sync.WaitGroup
	l           sync.Mutex
}

// Init logs in and initializes the agent
func (a *Agent) Init() error {
	if a.initialized {
		return fmt.Errorf("already initialized")
	}

	a.LoginUrl = "https://secure.emergencyreporting.com/"

	// Initialize all maps to avoid NPE
	a.reqMap = map[string]network.RequestID{}
	a.urlMap = map[string]string{}
	a.bodyMap = map[string][]byte{}
	a.attr = map[string]string{}
	a.cfunc = make([]context.CancelFunc, 0)
	a.done = make(chan string, 1)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(""),
		chromedp.Flag("enable-privacy-sandbox-ads-apis", true),
		chromedp.Flag("disable-web-security", true), // fix iframe issue?
	)

	lf := log.Printf
	if !a.Debug {
		lf = nil
	}

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	a.cfunc = append(a.cfunc, cancel)

	ctx, cancel = chromedp.NewContext(ctx, chromedp.WithDebugf(lf))
	a.cfunc = append(a.cfunc, cancel)

	/*
		// create a timeout as a safety net to prevent any infinite wait loops
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		a.cfunc = append(a.cfunc, cancel)
	*/

	// ensure that the browser process is started
	if err := chromedp.Run(ctx); err != nil {
		log.Printf("ERR: Run(): %s", err.Error())
		return err
	}

	// Listen to all network events and save content for whatever comes in
	chromedp.ListenTarget(ctx, func(v interface{}) {
		switch a.ContextSwitch {
		case ContextLogin:

			switch ev := v.(type) {
			case *network.EventRequestWillBeSent:
				//log.Printf("network.EventRequestWillBeSent")
				if unwantedTraffic(ev.Request.URL) {
					break
				}
				if a.Debug {
					log.Printf("EventRequestWillBeSent: %v: %v", ev.RequestID, ev.Request.URL)
				}
				a.l.Lock()
				a.reqMap[ev.Request.URL] = ev.RequestID
				a.l.Unlock()
			case *network.EventResponseReceived:
				//log.Printf("network.EventResponseReceived")
				if unwantedTraffic(ev.Response.URL) {
					break
				}

				if a.Debug {
					log.Printf("EventResponseReceived: %v: %v", ev.RequestID, ev.Response.URL)
					log.Printf("EventResponseReceived: status = %d, headers = %#v", ev.Response.Status, ev.Response.Headers)
				}
				a.l.Lock()
				a.urlMap[ev.RequestID.String()] = ev.Response.URL
				a.l.Unlock()
			case *network.EventLoadingFinished:
				//log.Printf("network.EventLoadingFinished")
				if a.Debug {
					log.Printf("EventLoadingFinished: %v", ev.RequestID)
				}
				a.wg.Add(1)
				go func() {
					c := chromedp.FromContext(ctx)
					body, err := network.GetResponseBody(ev.RequestID).Do(cdp.WithExecutor(ctx, c.Target))
					if err != nil {
						defer a.wg.Done()
						return
					}

					a.l.Lock()
					url := a.urlMap[ev.RequestID.String()]
					a.bodyMap[url] = body
					a.l.Unlock()

					if a.Debug {
						log.Printf("%s: %s", url, string(body))
					}

					defer a.wg.Done()
				}()
			}
			break
		case ContextDownload:
			if ev, ok := v.(*browser.EventDownloadProgress); ok {
				completed := "(unknown)"
				if ev.TotalBytes != 0 {
					completed = fmt.Sprintf("%0.2f%%", ev.ReceivedBytes/ev.TotalBytes*100.0)
				}
				log.Printf("state: %s, completed: %s\n", ev.State.String(), completed)
				if ev.State == browser.DownloadProgressStateCompleted {
					a.done <- ev.GUID
					close(a.done)
				}
			}
			break
		}
	})

	// Use a Chrome web browser to log in to the interface and obtain the
	// appropriate authentication token from local storage.

	if err := chromedp.Run(ctx,
		chromedp.Navigate(a.LoginUrl),
		chromedp.Tasks{
			// Login sequence
			//a.waitForLoadEvent(ctx),

			chromedp.ActionFunc(func(ctx context.Context) error {
				log.Printf("INFO: Attempting to load page")
				return nil
			}),

			chromedp.WaitVisible("//input[@data-test-id='usernameField']"),
			chromedp.SendKeys("//input[@data-test-id='usernameField']", a.Username),
			chromedp.SendKeys("//input[@data-test-id='passwordField']", a.Password),

			chromedp.ActionFunc(func(ctx context.Context) error {
				log.Printf("INFO: Attempting to submit form")
				return nil
			}),

			chromedp.Click("//button[@data-test-id='signInButton']"),

			chromedp.ActionFunc(func(ctx context.Context) error {
				log.Printf("INFO: Attempting to wait for dashboard to be visible")
				time.Sleep(5 * time.Second)
				return nil
			}),

			chromedp.ActionFunc(func(ctx context.Context) error {
				log.Printf("INFO: Loading base URL")
				return nil
			}),

			chromedp.Navigate(a.LoginUrl),

			// Don't continue until the dashboard is visible
			//chromedp.WaitVisible(`//*[contains(., 'Incidents')]`),

			chromedp.ActionFunc(func(ctx context.Context) error {
				log.Printf("INFO: Waiting until DIV.page-header-title is visible")
				return nil
			}),

			chromedp.WaitVisible("DIV.page-header-title"),

			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				a.cookies, err = network.GetCookies().Do(ctx)
				if err != nil {
					return err
				}
				for i, cookie := range a.cookies {
					if a.Debug {
						log.Printf("DEBUG: chrome cookie %d: %+v", i, cookie.Name)
					}
				}
				return nil
			}),

			/*
				chromedp.ActionFunc(func(ctx context.Context) error {
					cookies, err := network.GetCookies().Do(ctx)

					a.cookies = make([]network.Cookie, 0)
					var c []string
					for _, v := range cookies {
						a.cookies = append(a.cookies, *v)
						aCookie := v.Name + " - " + v.Domain
						c = append(c, aCookie)
					}

					stringSlices := strings.Join(c[:], ",\n")
					log.Printf("COOKIES: %v", stringSlices)

					if err != nil {
						return err
					}
					return nil
				}),
			*/

			/*
				chromedp.ActionFunc(func(ctx context.Context) error {
					log.Printf("INFO: Loading classes")
					return nil
				}),
				chromedp.Navigate("https://secure.emergencyreporting.com/training/classes.php"),

				chromedp.WaitVisible("//a[@id='addclass']"),

			*/

			/*
				chromedp.ActionFunc(func(ctx context.Context) error {

					// if the default profile is not loaded,
					// it just gets the entries added by the navigation action in the previous step.
					// it's possible that the js code to add cache entries is executed after this action,
					// and this action gets nothing.
					// in this case, it's better to listen to the DOMStorage events.
					log.Printf("INFO: Security Origin = %s", "https://"+strings.Split(a.LoginUrl, "/")[2])
					entries, err := domstorage.GetDOMStorageItems(&domstorage.StorageID{
						StorageKey:     domstorage.SerializedStorageKey("https://" + strings.Split(a.LoginUrl, "/")[2] + "/"),
						IsLocalStorage: true,
					}).Do(ctx)

					if err != nil {
						log.Printf("ERR: domstorage: %s", err.Error())
						return err
					}

					log.Printf("localStorage entries: %#v", entries)
						for _, entry := range entries {
							if strings.HasPrefix(entry[0], "oidc.user:") {
								//err = json.Unmarshal([]byte(entry[1]), &(a.auth))
								log.Printf("JSON user obj : %s", entry[1])
								if err != nil {
									log.Printf("ERR: Deserializing OIDC token: %s", err.Error())
								} else {
									log.Printf("INFO: oidc.expiresat = %d, oidc.auth_time = %d", a.auth.ExpiresAt, a.auth.Profile.AuthTime)
								}

							}
						}

					return nil
				}),
			*/

			/*
				chromedp.ActionFunc(func(ctx context.Context) error {
					log.Printf("INFO: Test agent-less login with cookies provided")

					cj, _ := cookiejar.New(nil)
					cookies := make([]*http.Cookie, 0)
					for _, v := range a.cookies {
						cookies = append(cookies, &http.Cookie{
							Name:    v.Name,
							Domain:  v.Domain,
							Expires: TimestampFromFloat64(v.Expires).Time,
						})
					}
					cj.SetCookies(shims.SingleValueDiscardError(url.Parse("https://secure.emergencyreporting.com")), cookies)

					client := http.Client{
						Jar: cj,
					}
					resp, err := client.Get("https://secure.emergencyreporting.com/nfirs/main.asp")
					if err != nil {
						return err
					}
					if resp.StatusCode > 399 {
						log.Printf("Response: %#v", resp)
					}

					body, _ := ioutil.ReadAll(resp.Body)
					log.Printf("BODY: %s", string(body))

					return nil
				}),
			*/
		},
	); err != nil {
		log.Printf("ERR: Failed to login: %s", err.Error())
		return err
	}

	if a.Debug {
		log.Printf("DEBUG: Wait for all data to be received.")
	}
	a.wg.Wait()

	if a.Debug {
		log.Printf("attr : %#v", a.attr)
		log.Printf("urlMap : %#v", a.urlMap)
	}

	if a.Debug {
		//	log.Printf("auth : %#v", a.auth)
	}

	a.initialized = true
	a.ctx = ctx

	return nil
}

func (a *Agent) Run() {
	go func() {
		for {
			if a.Debug {
				log.Printf("Run(): Ping()")
			}
			err := a.Ping()
			if err != nil {
				log.Printf("Run(): %s", err.Error())
			}
			for i := 0; i < 15; i++ {
				time.Sleep(time.Second)
				if a.cancelled {
					return
				}
			}
		}
	}()
}

func (a *Agent) Ping() error {
	return nil // TODO: FIXME: XXX
}

// authorizedGet uses the current authentication mechanism to GET a specific URL
func (a *Agent) authorizedGet(url string) ([]byte, error) {
	var out string

	log.Printf("authorizedGet(%s)", url)

	if err := chromedp.Run(a.ctx, chromedp.Navigate(url),
		chromedp.Tasks{
			chromedp.InnerHTML("//*", &out),
		}); err != nil {
		return nil, fmt.Errorf("could not get url %s: %s", url, err.Error())
	}
	return []byte(out), nil
}

// authorizedNativeGet uses the current authentication mechanism to GET a specific URL
func (a *Agent) authorizedNativeGet(url string) ([]byte, error) {
	var out []byte

	log.Printf("authorizedNativeGet(%s)", url)

	cl := http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("ERR: Parse: %s", err.Error())
		return []byte{}, err
	}

	// Load all cookies
	for _, i := range a.cookies {
		o := &http.Cookie{
			Name:    i.Name,
			Domain:  i.Domain,
			Value:   i.Value,
			Path:    i.Path,
			Expires: time.Now().Local().Add(time.Hour),
		}
		if a.Debug {
			log.Printf("DEBUG: cookie : %#v", *o)
		}
		req.AddCookie(o)
	}

	resp, err := cl.Do(req)
	if err != nil {
		log.Printf("ERR: Get: %s", err.Error())
		return []byte{}, err
	}

	out, err = io.ReadAll(resp.Body)

	return out, err
}

func (a *Agent) authorizedJsonGet(url string) ([]byte, error) {
	b, err := a.authorizedGet(url)
	if err != nil {
		return b, err
	}
	s := string(b)
	s = strings.TrimPrefix(s, "<head></head><body>")
	s = strings.TrimSuffix(s, "</body>")
	for {
		s = strings.TrimSuffix(s, "</span>")
		s = strings.TrimSuffix(s, "</div>")
		s = strings.TrimSuffix(s, "</a>")
		if !strings.HasSuffix(s, "</span>") && !strings.HasSuffix(s, "</div>") && !strings.HasSuffix(s, "</a>") {
			break
		}
	}
	s = strings.TrimSuffix(s, "</a>")

	s = strings.ReplaceAll(s, `\"=""`, `\"=\"\"`)
	s = strings.ReplaceAll(s, `\"=""`, `\"=\"\"`)

	b = []byte(s)
	return b, err
}

func (a *Agent) authorizedJsonGet2(url string) ([]byte, error) {
	var out string

	log.Printf("authorizedJsonGet2(%s)", url)

	if err := chromedp.Run(a.ctx, chromedp.Navigate(url),
		chromedp.Tasks{

			// Refresh cookies, keep 'em fresh so we don't die out during
			// enormous batches.
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				a.cookies, err = network.GetCookies().Do(ctx)
				if err != nil {
					return err
				}
				for i, cookie := range a.cookies {
					if a.Debug {
						log.Printf("DEBUG: chrome cookie %d: %+v", i, cookie.Name)
					}
				}
				return nil
			}),

			chromedp.Text(`//*`, &out),
		}); err != nil {
		return nil, fmt.Errorf("could not get url %s: %s", url, err.Error())
	}
	return []byte(out), nil
}

// authorizedApiGetCall accesses the api.emergencyreporting.com API, which
// requires tokens gleaned from the page. Rather than using the chromedp
// agent, we can use a standard native net/http request with the extracted
// access token. It won't work for a regular internal webservices call, as
// far as I can figure.
func (a *Agent) authorizedApiGetCall(hostPage, apiUrl string) ([]byte, error) {
	log.Printf("authorizedApiGetCall(%s, %s)", hostPage, apiUrl)

	var accessToken string
	if err := chromedp.Run(a.ctx,
		chromedp.Navigate(hostPage),
		chromedp.Evaluate(`$('#accessToken').val();`, &accessToken),
	); err != nil {
		return []byte{}, err
	}
	log.Printf("authorizedApiGetCall(): INFO: Got token %s", accessToken)

	// Basic fetch
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return []byte{}, err
	}
	req.Header.Set("Authorization", accessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (a *Agent) authorizedPost(url string, data map[string]any) ([]byte, error) {
	log.Printf("authorizedPost(%s)", url)

	js := `
async function postData(url = '', data = {}) {
  const response = await fetch(url, {
    method: 'POST',
    body: JSON.stringify(data)
  });
  return response.text();
}; postData('` + url + `', ` + string(shims.SingleValueDiscardError(json.Marshal(data))) + `)
    `

	var response string
	if err := chromedp.Run(a.ctx,
		chromedp.Evaluate(js, &response, func(ep *runtime.EvaluateParams) *runtime.EvaluateParams {
			return ep.WithAwaitPromise(true)
		}),
	); err != nil {
		return []byte{}, err
	}

	return []byte(response), nil
}

func (a *Agent) authorizedDownload(url string) (string, error) {
	return a.authorizedDownloadContext(a.ctx, url)
}

// authorizedDownload uses the current authentication mechanism to download a file.
// Returns the temporary file name.
func (a *Agent) authorizedDownloadContext(ctx context.Context, url string) (string, error) {
	var out string

	log.Printf("authorizedDownload(%s)", url)

	a.done = make(chan string, 1)
	a.ContextSwitch = ContextDownload

	wd := shims.SingleValueDiscardError(os.Getwd())

	if err := chromedp.Run(ctx,
		browser.
			SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(wd).
			WithEventsEnabled(true),
		chromedp.Navigate(url),
	); err != nil && !strings.Contains(err.Error(), "net::ERR_ABORTED") {
		log.Printf("authorizedDownload: ERR: %s", err.Error())
		return "", err
	}

	guid := <-a.done

	// We can predict the exact file location and name here because of how we
	// configured SetDownloadBehavior and WithDownloadPath
	out = filepath.Join(wd, guid)
	log.Printf("authorizedDownload: INFO: wrote %s", out)

	return out, nil
}

func (a *Agent) waitForLoadEvent(ctx context.Context) chromedp.Action {
	ch := make(chan struct{})

	lctx, cancel := context.WithCancel(ctx)
	go chromedp.ListenTarget(lctx, func(ev interface{}) {
		if _, ok := ev.(*page.EventLoadEventFired); ok {
			cancel()
			close(ch)
		}
	})

	return chromedp.ActionFunc(func(ctx context.Context) error {
		select {
		case <-ch:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func (a *Agent) getCsvUrl(csvurl string) ([][]string, error) {
	return a.getCsvUrlContext(a.ctx, csvurl)
}

func (a *Agent) getCsvUrlContext(ctx context.Context, csvurl string) ([][]string, error) {
	out := [][]string{}

	a.ContextSwitch = ContextDownload

	log.Printf("INFO: Load CSV from %s", csvurl)
	csvOut, err := a.authorizedDownload(csvurl)
	if err != nil {
		return out, err
	}

	log.Printf("INFO: CSV temporary file: %s", csvOut)
	defer os.Remove(csvOut)

	fp, err := os.Open(csvOut)
	if err != nil {
		return out, err
	}

	reader := csv.NewReader(fp)

	return reader.ReadAll()
}

type Timestamp struct {
	time.Time
}

func TimestampFromFloat64(ts float64) Timestamp {
	secs := int64(ts)
	nsecs := int64((ts - float64(secs)) * 1e9)
	return Timestamp{time.Unix(secs, nsecs)}
}

func getIframeContext(ctx context.Context, uriPart string) context.Context {
	targets, _ := chromedp.Targets(ctx)
	var tgt *target.Info
	for _, t := range targets {
		log.Printf("INFO: Frame %s | %s | %s | %#v", t.Title, t.Type, t.URL, t.TargetID)
		if (t.Type == "iframe" || t.Type == "frame") && strings.Contains(t.URL, uriPart) {
			log.Printf("INFO: Found target %#v", t)
			tgt = t
		}
	}
	if tgt != nil {
		ictx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(tgt.TargetID))
		return ictx
	}
	return nil
}
