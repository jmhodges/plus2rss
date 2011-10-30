include $(GOROOT)/src/Make.inc

TARG=plus2rss
GOFILES=\
	plus2rss.go\
	frontend.go\
	feed_store.go

include $(GOROOT)/src/Make.cmd
