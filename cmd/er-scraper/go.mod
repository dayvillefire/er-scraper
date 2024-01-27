module github.com/dayvillefire/er-scraper/cmd/er-scraper

go 1.21

toolchain go1.21.6

replace (
	github.com/dayvillefire/er-scraper => ../..
	github.com/dayvillefire/er-scraper/agent => ../../agent
)

require (
	github.com/dayvillefire/er-scraper/agent v0.0.0-20231227180914-bbdb912f7d34
	github.com/jbuchbinder/shims v0.0.0-20240127163204-18a2ea0be2dc
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/PuerkitoBio/goquery v1.8.1 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/chromedp/cdproto v0.0.0-20240127002248-bd7a66284627 // indirect
	github.com/chromedp/chromedp v0.9.3 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.3.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
)
