package dk

import (
	"math"
	"sort"
	"sync"
	"time"
)

type (
	Table struct {
		table          map[string]map[string]float64
		last_decay     time.Time
		me             *sync.Mutex
		decay_rate     float64
		decay_floor    float64
		decay_interval time.Duration
		running        bool
	}

	Report struct {
		Running     bool    `json:"running"`
		Timestamp   string  `json:"timestamp"`
		RenderTime  string  `json:"render_time"`
		TableSize   int     `json:"table_size"`
		DecayRate   float64 `json:"decay_rate"`
		DecayFloor  float64 `json:"decay_floor"`
		ResultCount int     `json:"result_count"`
		Results     Entries `json:"results"`
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

func NewTable(rate, floor float64, interval time.Duration) *Table {
	return &Table{
		table:          make(map[string]map[string]float64),
		me:             new(sync.Mutex),
		decay_rate:     rate,
		decay_floor:    floor,
		decay_interval: interval,
	}
}

func (d *Table) Start() {
	// fire up a goroutine to periodically decay the set
	d.last_decay = time.Now()
	d.running = true
	go func() {
		for d.running {
			if time.Since(d.last_decay) > d.decay_interval {
				d.me.Lock()
				d.decay()
				d.me.Unlock()
			}
			time.Sleep(d.decay_interval)
		}
	}()
}

func (d *Table) Stop() {
	d.running = false
}

func (d *Table) Reset() {
	d.me.Lock()
	d.table = make(map[string]map[string]float64)
	d.me.Unlock()
}

func (d *Table) Add(group, key string, inc float64) {

	d.me.Lock()
	if _, ok := d.table[group]; !ok {
		d.table[group] = make(map[string]float64)
	}
	d.table[group][key] += inc
	d.me.Unlock()

}

func (d *Table) decay() {

	// decay rate is applied as a compounding function
	dk_rate := math.Pow(1+d.decay_rate, float64(time.Since(d.last_decay).Nanoseconds())/float64(time.Second))

	d.last_decay = time.Now()

	for group, _ := range d.table {
		for name, value := range d.table[group] {

			// simple decay
			value /= dk_rate

			// clear out values that have decayed beyond relevance
			if value < d.decay_floor {
				delete(d.table[group], name)
			} else {
				d.table[group][name] = value
			}

		}
		if len(d.table[group]) == 0 {
			delete(d.table, group)
		}
	}

}
func (d *Table) Groups() (groups []string) {
	d.me.Lock()
	for name, _ := range d.table {
		groups = append(groups, name)
	}
	d.me.Unlock()
	return
}

func (d *Table) Report(g string, n int) *Report {

	start := time.Now()

	d.me.Lock()

	d.decay()

	set_size := len(d.table[g])

	// build a set of entries to sort and slice
	set := make(Entries, 0, set_size)

	for name, value := range d.table[g] {
		set = append(set, &Entry{name, value})
	}

	d.me.Unlock()

	// sort the values
	sort.Sort(set)

	// ensure n is between 1 and len(set)
	if n < 1 {
		n = 1
	} else if n > set_size {
		n = set_size
	}

	// reduce the set
	if n < len(set) {
		set = set[:n]
	}

	return &Report{
		Running:     d.running,
		Timestamp:   time.Now().String(),
		RenderTime:  time.Since(start).String(),
		TableSize:   set_size,
		DecayRate:   d.decay_rate,
		DecayFloor:  d.decay_floor,
		ResultCount: len(set),
		Results:     set,
	}
}
