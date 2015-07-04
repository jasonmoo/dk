package dk

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/jasonmoo/cardinal"
)

type (
	Table struct {
		sync.Mutex

		table map[string]*Column

		last_decay     time.Time
		decay_rate     float64
		decay_floor    float64
		decay_interval time.Duration
		running        bool
	}

	Column struct {
		values map[string]float64
		filter *cardinal.Cardinal
	}

	Report struct {
		Running    bool              `json:"running"`
		Timestamp  string            `json:"timestamp"`
		RenderTime string            `json:"render_time"`
		DecayRate  float64           `json:"decay_rate"`
		DecayFloor float64           `json:"decay_floor"`
		ResultSet  map[string]Result `json:"result_set"`
	}

	Result struct {
		TableSize   int         `json:"table_size"`
		Cardinality Cardinality `json:"cardinality"`
		ResultCount int         `json:"result_count"`
		Results     Entries     `json:"results"`
	}

	Cardinality struct {
		Percent  float64 `json:"percent"`
		Duration string  `json:"duration"`
		Uniques  uint64  `json:"uniques"`
		Total    uint64  `json:"total"`
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
		table:          make(map[string]*Column),
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
				d.Lock()
				d.decay()
				d.Unlock()
			}
			time.Sleep(d.decay_interval)
		}
	}()
}

func (d *Table) Stop() {
	d.running = false
}

func (d *Table) Reset() {
	d.Lock()
	d.table = make(map[string]*Column)
	d.Unlock()
}

func (d *Table) Add(column, key string, inc float64) {

	d.Lock()

	c, exists := d.table[column]

	if !exists {
		c = &Column{
			values: make(map[string]float64),
			filter: cardinal.New(time.Minute),
		}
		d.table[column] = c
	}

	c.values[key] += inc
	c.filter.Add(key)

	d.Unlock()

}

func (d *Table) decay() {

	// decay rate is applied as a compounding function
	dk_rate := math.Pow(1+d.decay_rate, float64(time.Since(d.last_decay).Nanoseconds())/float64(time.Second))

	d.last_decay = time.Now()

	for column, c := range d.table {
		for name, value := range c.values {

			// simple decay
			value /= dk_rate

			// clear out values that have decayed beyond relevance
			if value < d.decay_floor {
				delete(c.values, name)
			} else {
				c.values[name] = value
			}

		}
		if len(c.values) == 0 {
			delete(d.table, column)
		}
	}

}
func (d *Table) Columns() (columns []string) {
	d.Lock()
	for column, _ := range d.table {
		columns = append(columns, column)
	}
	d.Unlock()
	sort.Strings(columns)
	return
}
func (d *Table) ColumnCount() (n int) {
	d.Lock()
	n = len(d.table)
	d.Unlock()
	return
}
func (d *Table) KeyCount() (n int) {
	d.Lock()
	for _, c := range d.table {
		n += len(c.values)
	}
	d.Unlock()
	return
}

func (d *Table) Report(columns []string, n int) *Report {

	start, result_set := time.Now(), make(map[string]Result)

	if n < 1 {
		n = 1
	}

	d.Lock()

	d.decay()

	for _, column := range columns {

		c, exists := d.table[column]
		if !exists {
			continue
		}

		// build a set of entries to sort and slice
		set := make(Entries, 0, len(c.values))

		for name, value := range c.values {
			set = append(set, &Entry{name, value})
		}

		result_set[column] = Result{
			TableSize:   len(c.values),
			ResultCount: len(set),
			Cardinality: Cardinality{
				Duration: c.filter.Duration().String(),
				Total:    c.filter.Count(),
				Uniques:  c.filter.Uniques(),
				Percent:  c.filter.Cardinality(),
			},
			Results: set,
		}
	}

	d.Unlock()

	for g, set := range result_set {

		// sort the values
		sort.Sort(set.Results)

		// reduce the set
		if n < len(set.Results) {
			set.Results = set.Results[:n]
			set.ResultCount = n
		}

		result_set[g] = set

	}

	return &Report{
		Running:    d.running,
		Timestamp:  time.Now().String(),
		RenderTime: time.Since(start).String(),
		DecayRate:  d.decay_rate,
		DecayFloor: d.decay_floor,
		ResultSet:  result_set,
	}
}
