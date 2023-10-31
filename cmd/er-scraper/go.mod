module github.com/dayvillefire/er-scraper/cmd/er-scraper

go 1.20

replace (
	github.com/dayvillefire/er-scraper => ../..
	github.com/dayvillefire/er-scraper/agent => ../../agent
)

require (
	github.com/dayvillefire/er-scraper/agent v0.0.0-20231030220518-cb061555a090
	github.com/jbuchbinder/shims v0.0.0-20230728185230-53ce6a775b20
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/chromedp/cdproto v0.0.0-20231025043423-5615e204d422 // indirect
	github.com/chromedp/chromedp v0.9.3 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.3.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	golang.org/x/sys v0.13.0 // indirect
)
