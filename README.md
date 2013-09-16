
	       ._____
	     __| _/  | __
	    / __ ||  |/ /
	   / /_/ ||    <
	   \____ ||__|_ \
	        \/     \/

	        v 0,1


	dk is a decaying hashtable of counters.

	dk will track counts on keys in groups, and apply a constant time
	decay function to the set periodically to keep the data fresh.

	dk can be tuned from the command line: decay percentage per second,
	decay floor and decay minimum interval.


	Usage:

	// build and start server
	./build darwin/amd64 true && sudo bin/dk-darwin-amd64 -host :80


	// each call increments by one
	curl -s "http://localhost?g=users&k=jason" >> 200 OK
	curl -s "http://localhost?g=users&k=jason" >> 200 OK

	// or by v
	curl -s "http://localhost?g=users&k=jason&v=3" >> 200 OK

	// /top takes a group name and optional number of results
	curl -s "http://localhost/top?g=users&n=1" >>

	{
		index_size:  1,
		unix_nano:   1379281938905534700,
		render_time: "21.725us",
		decay_rate:  0.02,
		decay_floor: 0.5,
		results: [{
			name:  "jason",
			score: 3.5098815575373652
		}]
	}


#####HTTP Increment Options:
	http://localhost?g=users&k=jason&v=3

	g (group name)
	k (key name)
	v (optional; increment amount, defaults to 1)

#####HTTP Top Options:
	http://localhost/top?g=users&n=100

	g (group name)
	n (count to return; capped at 200)


#####Build Instructions:
The build script will pull down it's own copy of go to build with.  This can be manually reproduced by passing any second parameter to the build script.  The build script adds a few goodies at compile time.

	./build.sh linux/amd64 true   >> pulls down and builds for linux/amd64
	./build.sh darwin/amd64 true  >> pulls down and builds for darwin/amd64
	./build.sh darwin/amd64       >> rebuilds for darwin/amd64

	>> bin/dk-darwin-amd64  // binary output file

#####CLI Options:
Running the dk binary without any parameters will output full options

#####License
MIT
