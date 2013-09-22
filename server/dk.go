package main

import (
	dk ".."
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	table *dk.Table

	// cli options
	http_host      = flag.String("host", "", "addr:port to listen on for http")
	decay_rate     = flag.Float64("decay_rate", .02, "rate of decay per second")
	decay_floor    = flag.Float64("decay_floor", .5, "minimum value to keep")
	decay_interval = flag.Duration("decay_interval", 2*time.Second, "maximum amount to time between decays")
)

func group_list(host string) string {
	if groups := table.Groups(); len(groups) > 0 {
		glue := "\nhttp://" + host + "/top?g="
		return glue + strings.Join(groups, glue)
	}
	return "<no groups>"
}

func add_handler(w http.ResponseWriter, r *http.Request) {

	start := time.Now()

	g, k, v := r.FormValue("g"), r.FormValue("k"), r.FormValue("v")
	switch {
	case len(g) == 0, len(k) == 0:
		http.Error(w, fmt.Sprintf(web_usage, BuildInfo, group_list(r.Host)), http.StatusBadRequest)
		return
	}

	inc := 1.0
	if len(v) > 0 {
		inc, _ = strconv.ParseFloat(v, 64)
	}

	table.Add(g, k, inc)

	w.Header().Set("X-Render-Time", time.Since(start).String())
}

func top_n_handler(w http.ResponseWriter, r *http.Request) {

	start := time.Now()

	g := r.FormValue("g")
	if len(g) == 0 {
		http.Error(w, "Missing required field g (group name)\n\n"+group_list(r.Host), http.StatusBadRequest)
		return
	}

	n, _ := strconv.Atoi(r.FormValue("n"))
	if n < 1 {
		n = 10
	}

	h, report := w.Header(), table.Report(g, n)
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("X-Render-Time", time.Since(start).String())

	if err := json.NewEncoder(w).Encode(report); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func main() {

	log.Println("dk starting up")
	defer log.Println("dk exiting")

	flag.Parse()
	if flag.NFlag() < 1 {
		fmt.Println(usage)
		flag.PrintDefaults()
		fmt.Println()
		os.Exit(0)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	table = dk.NewTable(*decay_rate, *decay_floor, *decay_interval)
	table.Start()

	http.HandleFunc("/", add_handler)
	http.HandleFunc("/top", top_n_handler)

	log.Fatal(http.ListenAndServe(*http_host, nil))

}

const (
	usage = `
dk v0,1

dk will open an http endpoint for adding values / and querying top content /top

Notes:

  decay_rate and decay_floor allow you to set how aggressively you decay
  items from the set, and when to discard them.

  a higher floor will keep fewer items in memory but will keep
  fewer items in memory

  a higher decay_rate will make it harder for entries to survive
  where a lower one will keep the list populated

  decay_interval is a way to ensure the data set doesn't grow too big
  since we only decay it when it's being queried for topN ranges

Usage:
./dk -host :80 -decay_rate .002 -decay_floor 1 -decay_interval 2

Options:`

	// requires BuildInfo, group_list()
	web_usage = `Missing required fields:
g (group name)
k (key name)
v (optional; increment amount, defaults to 1)

%s

Group List:
%s
`
)
