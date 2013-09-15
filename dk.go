package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
)

type (
	Entry struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	}

	Entries []*Entry

	Report struct {
		IndexSize  int     `json:"index_size"`
		RenderTime string  `json:"render_time"`
		Results    Entries `json:"results"`
	}
)

func (e Entries) Len() int           { return len(e) }
func (e Entries) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e Entries) Less(i, j int) bool { return e[i].Score > e[j].Score }

var (
	// global index
	index map[string]map[string]float64

	// global last decay time
	last_decay = time.Now()

	// global read/write lock
	me = new(sync.Mutex)

	// cli options
	http_host      = flag.String("host", "", "addr:port to listen on for http")
	decay_rate     = flag.Float64("decay_rate", .02, "rate of decay per second")
	decay_floor    = flag.Float64("decay_floor", .5, "minimum value to keep")
	decay_interval = flag.Int("decay_interval", 10, "maximum number of seconds to go between decays")
)

const usage = `
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
./dk -decay_rate .002 -decay_floor 1

Options:`

func decay(rate, floor float64) {

	dk := math.Pow(1+rate, float64(time.Since(last_decay).Nanoseconds())/float64(time.Second))

	last_decay = time.Now()

	for group, _ := range index {
		for name, value := range index[group] {

			// simple decay
			value /= dk

			// clear out values that have decayed beyond relevance
			if value < floor {
				delete(index[group], name)
			} else {
				index[group][name] = value
			}

		}
		if len(index[group]) == 0 {
			delete(index, group)
		}
	}

}

func add_handler(w http.ResponseWriter, r *http.Request) {

	g, k, v := r.FormValue("g"), r.FormValue("k"), r.FormValue("v")
	if len(g) == 0 || len(k) == 0 {
		http.Error(w, "Missing required data g, k", http.StatusBadRequest)
		return
	}

	inc := 1.0
	if len(v) > 0 {
		inc, _ = strconv.ParseFloat(v, 64)
	}

	me.Lock()
	if _, ok := index[g]; !ok {
		index[g] = make(map[string]float64)
	}
	index[g][k] += inc
	me.Unlock()

}

func top_n_handler(w http.ResponseWriter, r *http.Request) {

	g := r.FormValue("g")
	if len(g) == 0 {
		http.Error(w, "Missing required data g", http.StatusBadRequest)
		return
	}

	n, _ := strconv.Atoi(r.FormValue("n"))
	if n < 1 {
		n = 10
	} else if n > 200 {
		n = 200
	}

	start := time.Now()

	me.Lock()

	decay(*decay_rate, *decay_floor)

	// build a set of entries to sort and slice
	set := make(Entries, 0, len(index[g]))

	for name, value := range index[g] {
		set = append(set, &Entry{name, value})
	}

	me.Unlock()

	// sort the values
	sort.Sort(set)

	// remove the min value
	if len(set) > n {
		set = set[:n]
	}

	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")

	json.NewEncoder(w).Encode(&Report{
		IndexSize:  len(index[g]),
		RenderTime: time.Since(start).String(),
		Results:    set,
	})

}

func init() {

	index = make(map[string]map[string]float64)

	// fire up a goroutine to periodically decay the set
	go func() {
		interval := time.Duration(*decay_interval) * time.Second
		for {
			<-time.After(interval)

			me.Lock()
			if time.Since(last_decay) > interval {
				decay(*decay_rate, *decay_floor)
			}
			me.Unlock()
		}
	}()

}

func main() {

	fmt.Println("dk starting up")
	defer fmt.Println("dk exiting")

	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()
	if flag.NFlag() < 1 {
		fmt.Println(usage)
		flag.PrintDefaults()
		fmt.Println()
		os.Exit(0)
	}

	http.HandleFunc("/", add_handler)
	http.HandleFunc("/top", top_n_handler)

	log.Fatal(http.ListenAndServe(*http_host, nil))

}
