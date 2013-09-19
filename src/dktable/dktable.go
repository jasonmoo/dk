package dktable

import (
	"math"
	"sort"
	"sync"
	"time"
)

type (
	DkTable struct {
		index          map[string]map[string]float64
		last_decay     time.Time
		me             *sync.Mutex
		decay_rate     float64
		decay_floor    float64
		decay_interval int
		running        bool
	}

	Report struct {
		Running    bool    `json:"running"`
		IndexSize  int     `json:"index_size"`
		Timestamp  int64   `json:"unix_nano"`
		RenderTime string  `json:"render_time"`
		DecayRate  float64 `json:"decay_rate"`
		DecayFloor float64 `json:"decay_floor"`
		Results    Entries `json:"results"`
	}

	Entries []*Entry

	Entry struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	}
)

func (e Entries) Len() int           { return len(e) }
func (e Entries) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e Entries) Less(i, j int) bool { return e[i].Score > e[j].Score }

func New(rate, floor float64, interval int) *DkTable {
	return &DkTable{
		index:          make(map[string]map[string]float64),
		me:             new(sync.Mutex),
		decay_rate:     rate,
		decay_floor:    floor,
		decay_interval: interval,
	}
}

func (d *DkTable) Init() {
	// fire up a goroutine to periodically decay the set
	d.last_decay = time.Now()
	d.running = true
	go func() {
		interval := time.Duration(d.decay_interval) * time.Second
		for d.running {
			if time.Since(d.last_decay) > interval {
				d.me.Lock()
				d.decay()
				d.me.Unlock()
			}
			time.Sleep(interval)
		}
	}()
}

func (d *DkTable) Shutdown() {
	d.me.Lock()
	d.running = false
	d.index = make(map[string]map[string]float64)
	d.me.Unlock()
}

func (d *DkTable) Reset() {
	d.me.Lock()
	d.index = make(map[string]map[string]float64)
	d.me.Unlock()
}

func (d *DkTable) Add(group, key string, inc float64) {

	d.me.Lock()
	if _, ok := d.index[group]; !ok {
		d.index[group] = make(map[string]float64)
	}
	d.index[group][key] += inc
	d.me.Unlock()

}

func (d *DkTable) decay() {

	// decay rate is applied as a compounding function
	dk := math.Pow(1+d.decay_rate, float64(time.Since(d.last_decay).Nanoseconds())/float64(time.Second))

	d.last_decay = time.Now()

	for group, _ := range d.index {
		for name, value := range d.index[group] {

			// simple decay
			value /= dk

			// clear out values that have decayed beyond relevance
			if value < d.decay_floor {
				delete(d.index[group], name)
			} else {
				d.index[group][name] = value
			}

		}
		if len(d.index[group]) == 0 {
			delete(d.index, group)
		}
	}

}
func (d *DkTable) Groups() (groups []string) {
	d.me.Lock()
	for name, _ := range d.index {
		groups = append(groups, name)
	}
	d.me.Unlock()
	return
}

func (d *DkTable) Report(g string, n int) *Report {

	start := time.Now()

	d.me.Lock()

	d.decay()

	group := d.index[g]

	// build a set of entries to sort and slice
	set, size := make(Entries, 0, len(group)), len(group)

	for name, value := range group {
		set = append(set, &Entry{name, value})
	}
	d.me.Unlock()

	// sort the values
	sort.Sort(set)

	// ensure n is between 1 and len(set)
	if n < 1 {
		n = 1
	} else if n > len(set) {
		n = len(set)
	}

	// reduce the set
	if n < len(set) {
		set = set[:n]
	}

	return &Report{
		Running:    d.running,
		IndexSize:  size,
		Timestamp:  time.Now().UnixNano(),
		RenderTime: time.Since(start).String(),
		DecayRate:  d.decay_rate,
		DecayFloor: d.decay_floor,
		Results:    set,
	}
}
