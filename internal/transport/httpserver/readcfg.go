package httpserver

import (
	"fmt"
	"net/http"
)

func readConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement q
	query := r.URL.Query()

	fmt.Println(query)

}
