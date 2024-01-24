package httpserver

import (
	"fmt"
	"net/http"
)

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement health check logic here
	fmt.Fprintln(w, "OK")
}
