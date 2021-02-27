package zplorama

import (
	"embed"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

//go:embed static/*
var staticContent embed.FS

func goToStatic(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/static/index.html", http.StatusMovedPermanently)
}

func staticFileHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(r.RequestURI))
}

func printRequest(request *http.Request) {
	fmt.Printf("%v %v %v\n", request.Method, request.URL, request.Proto)
	// Add the host
	fmt.Printf("Host: %v\n", request.Host)
	// Loop through headers
	for name, headers := range request.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			fmt.Printf("%v: %v\n", name, h)
		}
	}

	body, _ := ioutil.ReadAll(request.Body)
	fmt.Println(string(body))
}

// RunFrontendServer runs the server
func RunFrontendServer(port int, apiendpoint string) {
	serverMux := mux.NewRouter()

	serverMux.HandleFunc("/callback/", func(response http.ResponseWriter, request *http.Request) {
		printRequest(request)
		http.Redirect(response, request, "/static/", http.StatusMovedPermanently)
	})

	/*
		serverMux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
			printRequest(request)
			http.Redirect(response, request, "/static/", http.StatusMovedPermanently)
		})
	*/

	serverMux.PathPrefix("/").Handler(http.FileServer(http.FS(staticContent)))

	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%v", port),
		Handler: serverMux,
	}
	server.ListenAndServe()
}
