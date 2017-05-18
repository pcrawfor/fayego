package server

import (
	"fmt"
	"net/http"
)

func serveJSONP(f *Server, w http.ResponseWriter, r *http.Request) {
	jsonpParam := r.FormValue("jsonp")

	if len(jsonpParam) > 0 {
		if len(r.URL.Query()["message"]) < 1 {
			return
		}

		message := r.URL.Query()["message"][0]
		messageStr := fmt.Sprint(message)

		// handle the faye message
		response, err := f.HandleMessage([]byte(messageStr), nil)
		if err != nil {
			fmt.Println("Error parsing message: ", err)
		}

		finalResponse := jsonpParam + "(" + string(response) + ");"
		fmt.Println("THIS IS OUR HTTP RESPONSE: %v", finalResponse)
		fmt.Println("THIS IS THE W HEADERS: ", w.Header())
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, finalResponse)
		return
	}
}
