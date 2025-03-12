package scryfall

import (
	"strings"
)

type Card struct {
	Name    string     `json:"name"`
	ImgUris ImgURIs    `json:"image_uris"`
	Faces   []CardFace `json:"card_faces"`
}

type ImgURIs struct {
	Normal string `json:"normal"`
}
type CardFace struct {
	Name    string  `json:"name"`
	ImgUris ImgURIs `json:"image_uris"`
}

type MatchingFace struct {
	URL string
	ID  int64
}

func (sc Card) FindURL(name string) string {
	for _, f := range sc.Faces {
		if strings.EqualFold(f.Name, name) && f.ImgUris.Normal != "" {
			return f.ImgUris.Normal
		}
	}

	// fallback to top img
	if sc.ImgUris.Normal != "" {
		return sc.ImgUris.Normal
	}

	return ""
}
