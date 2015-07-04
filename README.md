
	       ._____
	     __| _/  | __
	    / __ ||  |/ /
	   / /_/ ||    <
	   \____ ||__|_ \
	        \/     \/

	        v 0,3

	a decaying 2-dimensional hashtable of counters

	  + a dk table tracks trends/cardinality on ephemeral data

	  + dk is optimized for fast writes and slower reads.

	  + dk was inspired by @bitly's https://github.com/bitly/forgettable

Used in several trend monitoring dashboards for high speed/throughput logs.

[DOCS](http://godoc.org/github.com/jasonmoo/dk)
[LICENSE](https://raw.github.com/jasonmoo/dk/master/LICENSE)
