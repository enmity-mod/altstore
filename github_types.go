package main

type header struct {
	Signature string `header:"x-hub-signature-256"`
}

type payload struct {
	Action  string  `json:"action"`
	Release release `json:"release"`
}

type release struct {
	Body      string  `json:"body"`
	CreatedAt string  `json:"created_at"`
	Assets    []asset `json:"assets"`
}

type asset struct {
	DownloadUrl string `json:"browser_download_url"`
	Name        string `json:"name"`
	Size        int    `json:"size"`
}
