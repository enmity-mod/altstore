package main

import (
	"archive/zip"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"howett.net/plist"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/update", func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, "https://altstore.enmity.app")
	})

	r.POST("/update", func(c *gin.Context) {
		// Get headers
		headers := header{}

		if err := c.ShouldBindHeader(&headers); err != nil {
			log.Println("couldn't parse headers")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// Get body
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Println("couldn't parse body")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Validate signature
		mac := hmac.New(sha256.New, []byte(os.Getenv("WEBHOOK_SECRET")))
		mac.Write([]byte(body))
		signatureString := fmt.Sprintf("sha256=%x", mac.Sum(nil))

		if signatureString != headers.Signature {
			log.Println("invalid signature")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Parse payload
		payload := payload{}
		json.Unmarshal([]byte(body), &payload)

		// Ignore non publish events
		if payload.Action != "published" {
			log.Println("action isn't published")
			c.AbortWithStatus(http.StatusOK)
			return
		}

		// Get our Enmity assets
		enmityStable, err := findAsset(payload.Release.Assets, "enmity.ipa")
		if err != nil {
			log.Println("enmity.ipa couldn't be found in release")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		enmityDev, err := findAsset(payload.Release.Assets, "enmity.dev.ipa")
		if err != nil {
			log.Println("enmity.dev.ipa couldn't be found in release")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Get the ipa version
		version, err := getVersionFromPlist(enmityStable.DownloadUrl)
		if err != nil {
			log.Println("couldn't get enmity's version")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Create our altstore entries
		stableApp := createAltstoreApp(enmityStable, &payload.Release, *version)
		devApp := createAltstoreApp(enmityDev, &payload.Release, *version)

		// Create our altstore release
		altstore, err := createAltstoreRelease([]app{*stableApp, *devApp})
		if err != nil {
			log.Println(err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Write the altstore repo
		if err := os.WriteFile(os.Getenv("ALTSTORE_FILE"), *altstore, 0666); err != nil {
			log.Println("couldn't write altstore file")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		log.Println("Altstore repo has been updated!")
		c.String(http.StatusOK, "Altstore repo has been updated!")
	})

	r.Run("127.0.0.1:8081")
}

// Find an asset by name
func findAsset(assets []asset, assetName string) (*asset, error) {
	for _, a := range assets {
		if a.Name == assetName {
			return &a, nil
		}
	}

	return nil, errors.New("not found")
}

// Get the final url
func getFinalURL(url string) (*string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.New("error getting url")
	}

	finalURL := resp.Request.URL.String()

	return &finalURL, nil
}

// Get the version string from info.plist
func getVersionFromPlist(url string) (*string, error) {
	// Get the real url
	finalURL, err := getFinalURL(url)
	if err != nil {
		return nil, err
	}

	// Get the file
	resp, err := http.Get(*finalURL)
	if err != nil {
		return nil, errors.New("couldn't get file")
	}

	defer resp.Body.Close()

	// Read the body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("couldn't read body")
	}

	// Read the ipa/zip
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}

	// Get our plist file
	var plistFile *zip.File
	for _, zipFile := range zipReader.File {
		if zipFile.Name == "Payload/Discord.app/Info.plist" {
			plistFile = zipFile
			break
		}
	}

	if plistFile == nil {
		return nil, errors.New("plist file missing from archive")
	}

	// Read our plistFile
	zfReader, err := plistFile.Open()
	if err != nil {
		return nil, err
	}

	defer zfReader.Close()

	plistContent, err := ioutil.ReadAll(zfReader)
	if err != nil {
		return nil, err
	}

	var info map[string]interface{}

	r := bytes.NewReader(plistContent)
	decoder := plist.NewDecoder(r)
	if err := decoder.Decode(&info); err != nil {
		return nil, err
	}

	// Get the version string
	version := info["CFBundleShortVersionString"].(string)
	return &version, nil
}

// Create our altsore release
func createAltstoreRelease(apps []app) (*[]byte, error) {
	// Read current release
	altstoreFile, err := ioutil.ReadFile(os.Getenv("ALTSTORE_FILE"))
	if err != nil {
		return nil, errors.New("couldn't read altstore file")
	}

	altstore := altstore{}
	json.Unmarshal(altstoreFile, &altstore)

	// Add our apps to the list
	altstore.Apps = append(apps, altstore.Apps...)

	// Create updated json
	updatedAltstore, err := json.MarshalIndent(altstore, "", "\t")
	if err != nil {
		return nil, errors.New("couldn't create new altstore json")
	}

	return &updatedAltstore, nil
}

// Creat an altstore app
func createAltstoreApp(asset *asset, release *release, version string) *app {
	app := app{
		Name:                 "Enmity",
		BundleIdentifier:     "com.hammerandchisel.discord",
		DeveloperName:        "Enmity Team",
		Subtitle:             "The power of addons, all in your hand.",
		Version:              version,
		VersionDate:          release.CreatedAt[0:10],
		VersionDescription:   release.Body,
		DownloadUrl:          asset.DownloadUrl,
		LocalizedDescription: "Add plugins and themes to Discord!",
		IconURL:              "https://files.enmity.app/icon-altstore.png",
		TintColor:            "6D00FF",
		Size:                 asset.Size,
		Beta:                 false,
	}

	return &app
}
