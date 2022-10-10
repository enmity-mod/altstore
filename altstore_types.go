package main

type altstore struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Apps       []app  `json:"apps"`
	UserInfo   struct {
	} `json:"userInfo"`
}

type app struct {
	Name                 string `json:"name"`
	BundleIdentifier     string `json:"bundleIdentifier"`
	DeveloperName        string `json:"developerName"`
	Subtitle             string `json:"subtitle"`
	Version              string `json:"version"`
	VersionDate          string `json:"versionDate"`
	VersionDescription   string `json:"versionDescription"`
	DownloadUrl          string `json:"downloadURL"`
	LocalizedDescription string `json:"localizedDescription"`
	IconURL              string `json:"iconURL"`
	TintColor            string `json:"tintColor"`
	Size                 int    `json:"size"`
	Beta                 bool   `json:"beta"`
}
