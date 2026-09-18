package repr

import (
	"regexp"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/imageclient"
)

type Image struct {
	URL       string  `json:"url" format:"uri" maxLength:"512" doc:"Absolute image URL. Never a bare hash."`
	Hash      string  `json:"hash" minLength:"64" maxLength:"64" pattern:"^[0-9a-f]{64}$" doc:"Image-service content hash."`
	Width     *int    `json:"width" minimum:"0" maximum:"65535" doc:"Pixel width. null if unknown."`
	Height    *int    `json:"height" minimum:"0" maximum:"65535" doc:"Pixel height. null if unknown."`
	Thumbhash *string `json:"thumbhash" maxLength:"128" pattern:"^[A-Za-z0-9+/=_-]+$" doc:"Thumbhash. null if unknown."`
	Sexual    *string `json:"sexual" enum:"safe,suggestive,explicit" maxLength:"11" doc:"Sexual depiction: safe, suggestive, or explicit. null means not assessed."`
}

var imageHash = regexp.MustCompile(`^[0-9a-f]{64}$`)

func NewImage(cdnBase, hash string, meta *imageclient.ImageMeta) *Image {
	if !imageHash.MatchString(hash) {
		return nil
	}
	return imageFrom(imageclient.MainURL(cdnBase, hash, "webp"), hash, meta)
}

// The first version returned the variant URL with the original's meta, so a _mini
// token (a 16:9 crop) carried the portrait original's width and height.
func NewImageFromToken(cdnBase, token string, meta *imageclient.ImageMeta) *Image {
	hash, _, ok := markdown.ParseContentImageRef(token)
	if !ok {
		return nil
	}
	return NewImage(cdnBase, hash, meta)
}

func imageFrom(url, hash string, meta *imageclient.ImageMeta) *Image {
	img := &Image{URL: url, Hash: hash, Sexual: sexualOf(meta)}
	if meta == nil {
		return img
	}
	if meta.Width > 0 {
		w := meta.Width
		img.Width = &w
	}
	if meta.Height > 0 {
		h := meta.Height
		img.Height = &h
	}
	if meta.Thumbhash != "" {
		th := meta.Thumbhash
		img.Thumbhash = &th
	}
	return img
}

func sexualOf(meta *imageclient.ImageMeta) *string {
	if meta == nil || meta.Sexual == nil {
		return nil
	}
	switch *meta.Sexual {
	case 0:
		s := "safe"
		return &s
	case 1:
		s := "suggestive"
		return &s
	case 2:
		s := "explicit"
		return &s
	default:
		return nil
	}
}
