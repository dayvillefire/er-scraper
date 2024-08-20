module github.com/dayvillefire/er-scraper/cmd/er-scraper

go 1.23

toolchain go1.23.0

replace (
	github.com/dayvillefire/er-scraper => ../..
	github.com/dayvillefire/er-scraper/agent => ../../agent
)

require (
	github.com/dayvillefire/er-scraper/agent v0.0.0-20240127175231-2a9c10659f74
	github.com/jbuchbinder/shims v0.0.0-20240506232043-4fac4ec97ccb
	github.com/joho/godotenv v1.5.1
)

require (
	github.com/PuerkitoBio/goquery v1.9.2 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/chromedp/cdproto v0.0.0-20240810084448-b931b754e476 // indirect
	github.com/chromedp/chromedp v0.10.0 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
)
