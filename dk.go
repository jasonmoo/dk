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
		Name  string
		Score float64
	}
	Entries []*Entry
)

func (e Entries) Len() int           { return len(e) }
func (e Entries) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e Entries) Less(i, j int) bool { return e[i].Score > e[j].Score }

var (
	// global index
	index = make(map[string]float64)

	last_decay = time.Now()

	// global read/write lock
	me = new(sync.Mutex)

	// soon
	// msgpack_host = flag.String("msgpack", ":81", "addr:port to listen on for msgpack rpc")

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

	for name, value := range index {

		// simple decay
		value /= dk

		// clear out values that have decayed beyond relevance
		if value < floor {
			delete(index, name)
		} else {
			index[name] = value
		}

	}

}

func add_handler(w http.ResponseWriter, r *http.Request) {

	k, v := r.FormValue("k"), r.FormValue("v")
	if len(k) == 0 {
		http.Error(w, "Missing required data k", http.StatusBadRequest)
		return
	}

	inc := 1.0
	if len(v) > 0 {
		inc, _ = strconv.ParseFloat(v, 64)
	}

	me.Lock()
	index[k] += inc
	me.Unlock()

}

func top_n_handler(w http.ResponseWriter, r *http.Request) {

	me.Lock()
	defer me.Unlock()

	start := time.Now()

	decay(*decay_rate, *decay_floor)

	n, _ := strconv.Atoi(r.FormValue("n"))
	if n < 1 {
		n = 10
	} else if n > 200 {
		n = 200
	}

	set := make(Entries, 0, n+1)

	for name, value := range index {
		// add the entry to the index
		set = append(set, &Entry{name, value})
		// sort the values
		sort.Sort(set)

		if len(set) > n {
			// remove the min value
			set = set[:n]
		}
	}

	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("X-Render-Time", time.Since(start).String())

	json.NewEncoder(w).Encode(set)

}

func init() {

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
