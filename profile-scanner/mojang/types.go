package mojang

type TextureResponse struct {
	Timestamp         int64  `json:"timestamp"`
	ProfileId         string `json:"profileId"`
	ProfileName       string `json:"profileName"`
	SignatureRequired bool   `json:"signatureRequired"`
	Textures          struct {
		Skin SkinURL `json:"SKIN"`
		Cape CapeURL `json:"CAPE"`
	} `json:"textures"`
}

type SkinURL struct {
	Url      string `json:"url"`
	Metadata struct {
		Model string `json:"model"`
	} `json:"metadata"`
}

type CapeURL struct {
	Url string `json:"url"`
}

type Document struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Textures Textures `json:"textures"`
}

type Textures struct {
	Skin Skin `json:"skin,omitempty"`
	Cape Cape `json:"cape,omitempty"`
}

type Skin struct {
	Data string `json:"data"`
}

type Cape struct {
	Data string `json:"data"`
}
