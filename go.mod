module github.com/pwiecz/portal_patterns

go 1.16

require (
	fyne.io/fyne/v2 v2.0.3
	github.com/chewxy/math32 v1.0.6
	github.com/golang/geo v0.0.0-20210211234256-740aa86cb551
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	golang.org/x/image v0.0.0-20210220032944-ac19c3e999fb
	golang.org/x/net v0.0.0-20210119194325-5f4716e94777 // indirect
)

//replace fyne.io/fyne/v2 v2.0.2 => github.com/pwiecz/fyne/v2 v2.0.4-0.20210503144308-1e916f84d24b
replace fyne.io/fyne/v2 v2.0.3 => ../fyne
