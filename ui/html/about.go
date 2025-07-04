package html

import (
	"time"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func AboutPage() Node {
	now := time.Now()

	return page("About",
		H1(Text("About")),

		P(Textf("Built with gomponents and rendered at %v.", now.Format(time.TimeOnly))),

		P(
			If(now.Second()%2 == 0, Text("It's an even second!")),
			If(now.Second()%2 != 0, Text("It's an odd second!")),
		),

		P(Textf("First pod name is: %s", GetEkswatches()[0].Synced)),

		// P(Textf("First pod name is: %s", "test-pod")),

		Img(Class("max-w-sm"), Src("https://www.gomponents.com/images/logo.png"), Alt("gomponents logo")),
	)
}
