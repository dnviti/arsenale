package emaildelivery

type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

type Status struct {
	Provider   string
	Configured bool
	From       string
}
