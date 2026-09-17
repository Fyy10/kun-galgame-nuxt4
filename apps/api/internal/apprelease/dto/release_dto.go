package dto

type AppDownloads struct {
	Android string `json:"android"`
	IOS     string `json:"ios"`
	Windows string `json:"windows"`
	Linux   string `json:"linux"`
}

type AppVersionResponse struct {
	MinVersion    string       `json:"min_version"`
	LatestVersion string       `json:"latest_version"`
	Notes         string       `json:"notes"`
	Downloads     AppDownloads `json:"downloads"`
}
