plus2rss
========

plus2rss is a small web server that provides Atom feeds of Google+ user
posts. Generating links to the Atom feeds is as easy as plugging a Google+
user's URL into the form on / of the server. Similarly, Google+ user ids can
be inputted.

plus2rss requires an API key for a Google account to be provided in a file
path. This file path is passed as `-simpleKeyFile` on the
command-line. plus2rss's other args can be seen with `plus2rss -h`.

Note that if you ship this thing to a server, you will need to bundle up the
`templates` directory and, if its location on the server is not in the same
directory as the executable, pass `-templateDir` to `plus2rss`.

You'll also want to adjust the `-vhost` parameter to match the public host
name (and optional port) you're serving traffic from. It's used to create
links internally.

(Finally, yep, plus2rss generates Atom, not RSS like it's name suggests. A
little white lie told for clarity.)
