package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONDecode(t *testing.T) {
	data := `{
		"id" : "e71be459ee504ec893dd0dfce4a5efd6",
		"name" : "mejoymenoy",
		"textures": {
			"skin": {
				"data": "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",
				"model": "slim"
			},
			"cape": {
				"data": "iVBORw0KGgoAAAANSUhEUgAAAEAAAAAgCAYAAACinX6EAAACiElEQVR4XuWWT2oVQRjE3xGCCDG4MFF0EzSIGFBBEZTss8km4M6NJ/AE7gR3HsO1kHU2LvUIHsATjKlmaqjU149umh5BevHj9avpmf6q+ps/m1enPzZnr3cmsH9wmIXHP9y4NX3Zuz99v3OUBcfmOZtfD55V8eTp2+nw4Yt0/tX/HOEcME1TFzZq0o3XHNuGF7wNBuB6CTfSyhLAp/e7qRCAolgYwDE3WMILLsF18fvm5Dyha/t8N9JK2gEWffF5PwQAjcd1bgkvuASvr0DT8FcLgIuh1dUwxmx/naegwFyHuMEaNHyupfXoreJGWkmL0sjHm7cTP789TvA/DXKumsdDD3gIbq4GnKfdB1DHqgEwZRj9und3MU2o6U64+VwIbq4GmsyZXy0ANU/4WlONIbDl3TxgtwA3V0MuAO0+0jUAFK1m9X3sOubSKMceAHU3VwON4tdvQT6guwegOw4u7x1fC0DBXDeaCwC4uRrY6t6VvCZ00D0AmNYAfj86SriuxeCrz0OARt3N1cCdVvPalRquG2llCYBmsRADwJjmNQAa5acvd1/1+VrBZAma59qsiVBzI61cC4DtzwDYBdAYkJrkbrt5MBccDFYQnkVO1wB0h9n+f14+T7ALAMb/IgCuo3B9rceNtJIC4IURAM0TLoixB0DzPQPgmtoBuVvBjbSSAuCiLQFsewbM5wWDJTQArWO1DsD9VhsA5qpJfw12CmAJget6ENDdSCtB6MVsIBgsoQFoEKt1gAu9mAsNBksggOnd+RIAxmSYDrgyu4SgATCEIQLA+blb4L/pgPm1FQyWkA5YQvDd7/oh5MJoBGE0gjAaQRiNIIxGEEYjCKMRhNEIwmgEYTT+AksiJ9gdoefNAAAAAElFTkSuQmCC"
			}
		}  
	}`

	obj := Document{}

	if err := json.NewDecoder(strings.NewReader(data)).Decode(&obj); err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", obj)
	t.Logf("%#v", obj.Textures.Skin)
}

func TestJSONEncode(t *testing.T) {
	obj := Document{
		Id:   "e71be459ee504ec893dd0dfce4a5efd6",
		Name: "mejoymenoy",
		Textures: Textures{
			Skin: &Skin{
				Model: "",
			},
		},
	}

	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(obj); err != nil {
		t.Fatal(err)
	}

	t.Log(buf.String())
}
