package httpserver

import (
	"fmt"
	"net/http"
)

func readinessCheckHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: implement readiness check logic here
	fmt.Fprintln(w, "OK")
}
