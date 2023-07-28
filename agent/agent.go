package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/jbuchbinder/shims"
)

var (
	ErrNotAuthorized = errors.New("not authorized")
)

type Agent struct {
	Debug    bool
	LoginUrl string
	Username string
	Password string

	reqMap  map[string]network.RequestID
	urlMap  map[string]string
	bodyMap map[string][]byte
	attr    map[string]string
	cookies []network.Cookie
	ctx     context.Context
	cfunc   []context.CancelFunc

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

	// Initialize all maps to avoid NPE
	a.reqMap = map[string]network.RequestID{}
	a.urlMap = map[string]string{}
	a.bodyMap = map[string][]byte{}
	a.attr = map[string]string{}
	a.cfunc = make([]context.CancelFunc, 0)

	var _ctx context.Context
	var _cancel context.CancelFunc

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(""),
		chromedp.Flag("enable-privacy-sandbox-ads-apis", true),
	)

	_ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	a.cfunc = append(a.cfunc, cancel)

	if a.Debug {
		_ctx, _cancel = chromedp.NewContext(
			_ctx,
			chromedp.WithDebugf(log.Printf),
		)
	} else {
		_ctx, _cancel = chromedp.NewContext(
			_ctx,
		)
	}
	a.cfunc = append(a.cfunc, _cancel)

	ctx, cancel := context.WithTimeout(_ctx, 60*time.Second)
	a.cfunc = append(a.cfunc, cancel)

	// ensure that the browser process is started
	if err := chromedp.Run(ctx); err != nil {
		log.Printf("ERR: Run(): %s", err.Error())
		return err
	}

	// Listen to all network events and save content for whatever comes in
	chromedp.ListenTarget(ctx, func(v interface{}) {
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

			chromedp.Navigate(a.LoginUrl),

			// Don't continue until the dashboard is visible
			//chromedp.WaitVisible(`//*[contains(., 'Incidents')]`),
			chromedp.WaitVisible("DIV.page-header-title"),

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

// authorizedDownload uses the current authentication mechanism to download a file
func (a *Agent) authorizedDownload(url string) ([]byte, error) {
	var out []byte

	log.Printf("authorizedDownload(%s)", url)

	var requestID network.RequestID
	done := make(chan string, 1)

	_ctx, _cancel := context.WithCancel(a.ctx)

	// set up a listener to watch the network events and close the channel when
	// complete the request id matching is important both to filter out
	// unwanted network events and to reference the downloaded file later
	chromedp.ListenTarget(_ctx, func(v interface{}) {
		if ev, ok := v.(*browser.EventDownloadProgress); ok {
			completed := "(unknown)"
			if ev.TotalBytes != 0 {
				completed = fmt.Sprintf("%0.2f%%", ev.ReceivedBytes/ev.TotalBytes*100.0)
			}
			log.Printf("state: %s, completed: %s\n", ev.State.String(), completed)
			if ev.State == browser.DownloadProgressStateCompleted {
				done <- ev.GUID
				close(done)
			}
		}
	})

	wd := shims.SingleValueDiscardError(os.Getwd())

	if err := chromedp.Run(a.ctx,
		browser.
			SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(wd).
			WithEventsEnabled(true),
		chromedp.Navigate(url),
	); err != nil {
		if _cancel != nil {
			_cancel()
		}
		log.Printf("authorizedDownload: ERR: %s", err.Error())
		return []byte{}, err
	}

	guid := <-done

	// We can predict the exact file location and name here because of how we
	// configured SetDownloadBehavior and WithDownloadPath
	log.Printf("wrote %s", filepath.Join(wd, guid+".zip"))

	if _cancel != nil {
		_cancel()
	}

	var err error
	if err = chromedp.Run(a.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		out, err = network.GetResponseBody(requestID).Do(ctx)
		return err
	})); err != nil {
		return []byte{}, err
	}
	if len(out) < 1 {
		return []byte{}, fmt.Errorf("no data")
	}

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

type Timestamp struct {
	time.Time
}

func TimestampFromFloat64(ts float64) Timestamp {
	secs := int64(ts)
	nsecs := int64((ts - float64(secs)) * 1e9)
	return Timestamp{time.Unix(secs, nsecs)}
}
