package main

import (
  "context"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "sort"
  "strings"

  "github.com/go-chi/chi"
  "github.com/go-chi/chi/middleware"
)

var HTTP_BIND = ":8000"

func main() {
  bind := _env("HTTP_BIND", HTTP_BIND)
  mux  := chi.NewRouter()

  mux.Use(middleware.Logger)
  mux.Use(remoteAddr)

  mux.Get("/healthz", http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {w.Write([]byte("healthy\n"));}));
  mux.Post("/hooker", hooker())

  if token := os.Getenv("NODES_AUTHORIZATION"); token != "" {
    log.Printf("util: nodes: enabled (token=%s)", token)
    mux.Mount("/nodes", nodesRouter(token))
  } else {
    log.Printf("util: nodes: disabled")
  }

  log.Printf("http: bind: %s", bind)
  err := http.ListenAndServe(bind, mux)
  if err != nil {
    log.Fatalf("http error=%v", err)
  }
}

func hooker() http.HandlerFunc {
  return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
      log.Printf("hooker: read: error: %v", err)
      _err(w, http.StatusInternalServerError)
      return
    }

    log.Printf("hooker: headers")
    names := make([]string, 0, len(r.Header))
    for name, _ := range r.Header {
      names = append(names, name)
    }
    sort.Strings(names)
    for _, name := range names {
      log.Printf("  %s: %s", name, strings.Join(r.Header[name], ", "))
    }

    log.Printf("hooker: body (%d bytes)", len(body))
    log.Printf("%s", string(body))

    switch r.Header.Get("Content-Type") {
    case "application/json":
      var object interface{}
      if err := json.Unmarshal(body, &object); err != nil {
        log.Printf("hooker: json: unmarshal error: %v", err)
      } else if text, err := json.MarshalIndent(object, "", "  "); err != nil {
        log.Printf("hooker: json: marshal error: %v", err)
      } else {
        log.Printf("hooker: json")
        log.Printf("%s", text)
      }
    }

    w.Write([]byte("200 OK\r\n"))
  })
}

func nodesRouter(token string) http.Handler {
  store  := map[string] string{}
  router := chi.NewRouter()
  router.Get("/", nodes(token, store))
  router.Put("/", nodes(token, store))
  return router
}

func nodes(token string, store map[string] string) http.HandlerFunc {
  bearer := "Bearer " + token
  return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
    if r.Header.Get("Authorization") != bearer {
      _err(w, http.StatusForbidden)
      return
    }

    switch r.Method {
    case "GET":
      if name := r.FormValue("name"); name == "" {
        _err(w, http.StatusBadRequest)
      } else if ip, found := store[name]; !found {
        _err(w, http.StatusNotFound)
      } else {
        w.Write([]byte(ip + "\r\n"))
      }

    case "PUT":
      if name := r.FormValue("name"); name == "" {
        _err(w, http.StatusBadRequest)
      } else {
        store[name] = r.Context().Value("RemoteAddr").(string);
      }

    default:
      _err(w, http.StatusForbidden)
    }

    return
  })
}

func remoteAddr(next http.Handler) http.Handler {
  return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
    remoteAddr := r.RemoteAddr
    forwardFor := r.Header.Get("X-Forwarded-For")
    if forwardFor != "" {
      remoteAddr = strings.Split(forwardFor, ", ")[0]
    }
    index := strings.LastIndex(remoteAddr, ":")
    if index >= 0 {
      remoteAddr = remoteAddr[0:index]
    }
    next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "RemoteAddr", remoteAddr)))
  })
}

func _env(name, ifEmpty string) string {
  value := os.Getenv(name)
  if value == "" {
    value = ifEmpty
  }
  return value
}

func _err(w http.ResponseWriter, code int) {
  http.Error(w, fmt.Sprintf("%d %s", code, http.StatusText(code)), code);
}
