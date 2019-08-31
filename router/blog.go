package router

import (
	"github.com/benitogf/katamari/messages"
	"net/http"
	"github.com/benitogf/katamari/objects"
	"encoding/json"
	"github.com/benitogf/katamari"
)

func blogFilter(index string, data []byte) ([]byte, error) {
	type post struct {
		Active bool `json:"active"`
	}
	unfiltered, err := objects.DecodeList(data)
	if err != nil {
		return []byte(""), err
	}
	filtered := []objects.Object{}
	for _, obj := range unfiltered {
		var postData post
		err = json.Unmarshal([]byte(obj.Data), &postData)
		if err == nil && postData.Active {
			obj.Data = messages.Encode([]byte(obj.Data))
			filtered = append(filtered, obj)
		}
	}
	rawFiltered, err := objects.Encode(filtered)
	if err != nil {
		return []byte(""), err
	}
	return rawFiltered, nil
}

func blogStream(server *katamari.Server, w http.ResponseWriter, r *http.Request) {
	client, err := server.Stream.New("posts/*", "blog", w, r)
	if err != nil {
		return
	}

	entry, err := server.Fetch("posts/*", "blog")
	if err != nil {
		return
	}

	go server.Stream.Write(client, messages.Encode(entry.Data), true, entry.Version)
	server.Stream.Read("posts/*", "blog", client)
}